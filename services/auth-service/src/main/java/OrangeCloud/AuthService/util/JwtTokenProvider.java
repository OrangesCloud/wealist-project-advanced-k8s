package OrangeCloud.AuthService.util;

import OrangeCloud.AuthService.exception.CustomJwtException;
import OrangeCloud.AuthService.exception.ErrorCode;
import io.jsonwebtoken.*;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;

import java.security.interfaces.RSAPrivateKey;
import java.security.interfaces.RSAPublicKey;
import java.util.Date;
import java.util.HashMap;
import java.util.Map;
import java.util.UUID;

/**
 * JWT 토큰 생성 및 검증
 * RS256 (RSA + SHA-256) 알고리즘 사용
 */
@Component
public class JwtTokenProvider {

    private static final Logger logger = LoggerFactory.getLogger(JwtTokenProvider.class);

    private final RSAPublicKey publicKey;
    private final RSAPrivateKey privateKey;
    private final String keyId;
    private final String issuer;
    private final long accessTokenExpirationMs;
    private final long refreshTokenExpirationMs;

    public JwtTokenProvider(
            RSAPublicKey rsaPublicKey,
            RSAPrivateKey rsaPrivateKey,
            @Value("${jwt.rsa.key-id:wealist-auth-key-1}") String keyId,
            @Value("${jwt.issuer:wealist-auth-service}") String issuer,
            @Value("${jwt.access-token-expiration-ms:1800000}") long accessTokenExpirationMs,
            @Value("${jwt.refresh-token-expiration-ms:604800000}") long refreshTokenExpirationMs) {

        this.publicKey = rsaPublicKey;
        this.privateKey = rsaPrivateKey;
        this.keyId = keyId;
        this.issuer = issuer;
        this.accessTokenExpirationMs = accessTokenExpirationMs;
        this.refreshTokenExpirationMs = refreshTokenExpirationMs;

        logger.info("JwtTokenProvider initialized with RS256 algorithm, issuer: {}, keyId: {}", issuer, keyId);
    }

    /**
     * Access Token 생성 (RS256)
     */
    public String generateToken(UUID userId) {
        Date now = new Date();
        Date expiryDate = new Date(now.getTime() + accessTokenExpirationMs);

        Map<String, Object> header = new HashMap<>();
        header.put("typ", "JWT");
        header.put("alg", "RS256");
        header.put("kid", keyId);

        return Jwts.builder()
                .setHeader(header)
                .setSubject(userId.toString())
                .setIssuer(issuer)
                .setIssuedAt(now)
                .setExpiration(expiryDate)
                .claim("type", "access")
                .signWith(privateKey, SignatureAlgorithm.RS256)
                .compact();
    }

    /**
     * Refresh Token 생성 (RS256)
     */
    public String generateRefreshToken(UUID userId) {
        Date now = new Date();
        Date expiryDate = new Date(now.getTime() + refreshTokenExpirationMs);

        Map<String, Object> header = new HashMap<>();
        header.put("typ", "JWT");
        header.put("alg", "RS256");
        header.put("kid", keyId);

        return Jwts.builder()
                .setHeader(header)
                .setSubject(userId.toString())
                .setIssuer(issuer)
                .setIssuedAt(now)
                .setExpiration(expiryDate)
                .claim("type", "refresh")
                .signWith(privateKey, SignatureAlgorithm.RS256)
                .compact();
    }

    /**
     * Token 유효성 검사
     */
    public void validateToken(String token) {
        try {
            Jwts.parserBuilder()
                .setSigningKey(publicKey)
                .build()
                .parseClaimsJws(token);
        } catch (io.jsonwebtoken.security.SignatureException e) {
            logger.error("Invalid JWT signature: {}", e.getMessage());
            throw new CustomJwtException(ErrorCode.TOKEN_SIGNATURE_INVALID);
        } catch (MalformedJwtException e) {
            logger.error("Invalid JWT token: {}", e.getMessage());
            throw new CustomJwtException(ErrorCode.MALFORMED_TOKEN);
        } catch (ExpiredJwtException e) {
            logger.error("Expired JWT token: {}", e.getMessage());
            throw new CustomJwtException(ErrorCode.EXPIRED_TOKEN);
        } catch (UnsupportedJwtException e) {
            logger.error("Unsupported JWT token: {}", e.getMessage());
            throw new CustomJwtException(ErrorCode.UNSUPPORTED_TOKEN);
        } catch (IllegalArgumentException e) {
            logger.error("JWT claims string is empty: {}", e.getMessage());
            throw new CustomJwtException(ErrorCode.INVALID_TOKEN);
        }
    }

    /**
     * Token에서 사용자 ID 추출
     */
    public UUID getUserIdFromToken(String token) {
        try {
            Claims claims = Jwts.parserBuilder()
                    .setSigningKey(publicKey)
                    .build()
                    .parseClaimsJws(token)
                    .getBody();
            return UUID.fromString(claims.getSubject());
        } catch (Exception e) {
            throw new CustomJwtException(ErrorCode.INVALID_TOKEN);
        }
    }

    /**
     * Token 만료 시간 가져오기
     */
    public Date getExpirationDateFromToken(String token) {
        try {
            Claims claims = Jwts.parserBuilder()
                    .setSigningKey(publicKey)
                    .build()
                    .parseClaimsJws(token)
                    .getBody();
            return claims.getExpiration();
        } catch (ExpiredJwtException e) {
            return e.getClaims().getExpiration();
        } catch (Exception e) {
            throw new CustomJwtException(ErrorCode.INVALID_TOKEN);
        }
    }

    /**
     * 토큰 타입 확인 (access/refresh)
     */
    public String getTokenType(String token) {
        try {
            Claims claims = Jwts.parserBuilder()
                    .setSigningKey(publicKey)
                    .build()
                    .parseClaimsJws(token)
                    .getBody();
            return claims.get("type", String.class);
        } catch (Exception e) {
            return null;
        }
    }

    /**
     * RSA 공개키 반환 (JWKS 엔드포인트용)
     */
    public RSAPublicKey getPublicKey() {
        return publicKey;
    }

    /**
     * Key ID 반환
     */
    public String getKeyId() {
        return keyId;
    }

    /**
     * Issuer 반환
     */
    public String getIssuer() {
        return issuer;
    }
}
