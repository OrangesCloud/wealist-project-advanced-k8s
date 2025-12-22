package OrangeCloud.AuthService.controller;

import OrangeCloud.AuthService.util.JwtTokenProvider;
import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.tags.Tag;
import org.springframework.http.MediaType;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

import java.math.BigInteger;
import java.security.interfaces.RSAPublicKey;
import java.util.Base64;
import java.util.List;
import java.util.Map;

/**
 * JWKS (JSON Web Key Set) 엔드포인트
 * Istio, API Gateway 등에서 JWT 검증에 사용
 */
@Tag(name = "JWKS", description = "JSON Web Key Set 엔드포인트")
@RestController
public class JwksController {

    private final JwtTokenProvider jwtTokenProvider;

    public JwksController(JwtTokenProvider jwtTokenProvider) {
        this.jwtTokenProvider = jwtTokenProvider;
    }

    /**
     * JWKS 엔드포인트 - RFC 7517 형식
     * URL: /.well-known/jwks.json
     */
    @Operation(summary = "JWKS 조회", description = "JWT 검증용 공개키 세트 반환")
    @GetMapping(value = "/.well-known/jwks.json", produces = MediaType.APPLICATION_JSON_VALUE)
    public Map<String, Object> getJwks() {
        RSAPublicKey publicKey = jwtTokenProvider.getPublicKey();
        String keyId = jwtTokenProvider.getKeyId();

        // RSA 공개키를 JWK 형식으로 변환
        Map<String, Object> jwk = Map.of(
            "kty", "RSA",
            "use", "sig",
            "alg", "RS256",
            "kid", keyId,
            "n", base64UrlEncode(publicKey.getModulus()),
            "e", base64UrlEncode(publicKey.getPublicExponent())
        );

        return Map.of("keys", List.of(jwk));
    }

    /**
     * OpenID Connect Discovery 엔드포인트 (선택적)
     * URL: /.well-known/openid-configuration
     */
    @Operation(summary = "OpenID Configuration", description = "OpenID Connect Discovery 메타데이터")
    @GetMapping(value = "/.well-known/openid-configuration", produces = MediaType.APPLICATION_JSON_VALUE)
    public Map<String, Object> getOpenIdConfiguration() {
        String issuer = jwtTokenProvider.getIssuer();

        // 기본 issuer URL 구성 (실제 환경에서는 환경변수로 설정 권장)
        String issuerUrl = "http://auth-service:8080";

        return Map.of(
            "issuer", issuer,
            "jwks_uri", issuerUrl + "/.well-known/jwks.json",
            "authorization_endpoint", issuerUrl + "/oauth2/authorization/google",
            "token_endpoint", issuerUrl + "/api/auth/refresh",
            "userinfo_endpoint", issuerUrl + "/api/auth/me",
            "response_types_supported", List.of("code"),
            "subject_types_supported", List.of("public"),
            "id_token_signing_alg_values_supported", List.of("RS256"),
            "scopes_supported", List.of("openid", "email", "profile")
        );
    }

    /**
     * BigInteger를 Base64 URL 인코딩
     */
    private String base64UrlEncode(BigInteger value) {
        byte[] bytes = value.toByteArray();

        // BigInteger는 부호 비트로 인해 앞에 0x00이 붙을 수 있음 - 제거
        if (bytes[0] == 0 && bytes.length > 1) {
            byte[] tmp = new byte[bytes.length - 1];
            System.arraycopy(bytes, 1, tmp, 0, tmp.length);
            bytes = tmp;
        }

        return Base64.getUrlEncoder().withoutPadding().encodeToString(bytes);
    }
}
