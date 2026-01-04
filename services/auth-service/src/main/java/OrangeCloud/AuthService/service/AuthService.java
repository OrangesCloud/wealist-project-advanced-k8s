package OrangeCloud.AuthService.service;

import OrangeCloud.AuthService.dto.AuthResponse;
import OrangeCloud.AuthService.exception.CustomJwtException;
import OrangeCloud.AuthService.exception.ErrorCode;
import OrangeCloud.AuthService.util.JwtTokenProvider;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.stereotype.Service;

import java.time.Duration;
import java.util.Date;
import java.util.UUID;

@Service
@RequiredArgsConstructor
@Slf4j
public class AuthService {

    private final JwtTokenProvider tokenProvider;
    private final RedisTemplate<String, Object> redisTemplate;

    // ============================================================================
    // 토큰 발행
    // ============================================================================

    /**
     * 사용자 ID로 Access Token과 Refresh Token 발행
     */
    public AuthResponse generateTokens(UUID userId) {
        return generateTokens(userId, null);
    }

    /**
     * 사용자 ID와 이메일로 Access Token과 Refresh Token 발행
     * ops-portal 등 email 기반 인증이 필요한 서비스용
     */
    public AuthResponse generateTokens(UUID userId, String email) {
        log.debug("Generating tokens for user: {}, email: {}", userId, email);

        String accessToken = tokenProvider.generateToken(userId, email);
        String refreshToken = tokenProvider.generateRefreshToken(userId, email);

        return new AuthResponse(accessToken, refreshToken, userId);
    }

    // ============================================================================
    // 로그아웃
    // ============================================================================

    /**
     * 로그아웃 - 토큰을 Redis 블랙리스트에 추가
     */
    public void logout(String token) {
        log.debug("Attempting to log out token");

        tokenProvider.validateToken(token);

        Date expirationDate = tokenProvider.getExpirationDateFromToken(token);
        long ttl = expirationDate.getTime() - System.currentTimeMillis();

        if (ttl > 0) {
            redisTemplate.opsForValue().set(token, "blacklisted", Duration.ofMillis(ttl));
            log.debug("Token blacklisted successfully in Redis with TTL: {}ms", ttl);
        } else {
            log.warn("Token is already expired. Not adding to blacklist");
        }
    }

    // ============================================================================
    // 토큰 갱신
    // ============================================================================

    /**
     * Refresh Token을 사용하여 새로운 Access Token 발급
     * email claim이 있으면 새 토큰에도 포함
     */
    public AuthResponse refreshToken(String refreshToken) {
        log.debug("Attempting to refresh token");

        tokenProvider.validateToken(refreshToken);

        if (isTokenBlacklisted(refreshToken)) {
            log.warn("Refresh token is blacklisted");
            throw new CustomJwtException(ErrorCode.TOKEN_BLACKLISTED);
        }

        UUID userId = tokenProvider.getUserIdFromToken(refreshToken);
        String email = tokenProvider.getEmailFromToken(refreshToken);
        log.debug("Extracted user ID: {}, email: {} from refresh token", userId, email);

        // 기존 refresh token 블랙리스트 추가
        Date expirationDate = tokenProvider.getExpirationDateFromToken(refreshToken);
        long ttl = expirationDate.getTime() - System.currentTimeMillis();
        if (ttl > 0) {
            redisTemplate.opsForValue().set(refreshToken, "blacklisted", Duration.ofMillis(ttl));
            log.debug("Old refresh token blacklisted with TTL: {}ms", ttl);
        }

        // 새로운 토큰 생성 (email 포함)
        String newAccessToken = tokenProvider.generateToken(userId, email);
        String newRefreshToken = tokenProvider.generateRefreshToken(userId, email);

        return new AuthResponse(newAccessToken, newRefreshToken, userId);
    }

    // ============================================================================
    // 토큰 유효성 검증 (외부 서비스용)
    // ============================================================================

    /**
     * Access Token의 유효성을 검증하고 사용자 ID를 반환합니다.
     */
    public UUID validateTokenAndGetUserId(String token) {
        log.debug("Validating token for external service use.");

        // 1. 토큰 유효성 검사 (서명, 만료 시간 확인)
        tokenProvider.validateToken(token);

        // 2. 토큰이 블랙리스트에 있는지 확인 (로그아웃된 토큰인지 확인)
        if (isTokenBlacklisted(token)) {
            log.warn("Attempted to use a blacklisted token.");
            throw new CustomJwtException(ErrorCode.TOKEN_BLACKLISTED);
        }

        // 3. 토큰에서 User ID 추출
        UUID userId = tokenProvider.getUserIdFromToken(token);
        log.info("Token validated successfully, user ID: {}", userId);

        return userId;
    }

    // ============================================================================
    // 토큰 블랙리스트 확인
    // ============================================================================

    /**
     * 토큰이 Redis 블랙리스트에 있는지 확인
     */
    public boolean isTokenBlacklisted(String token) {
        log.debug("Checking if token is blacklisted");
        Boolean isBlacklisted = redisTemplate.hasKey(token);
        return Boolean.TRUE.equals(isBlacklisted);
    }
}
