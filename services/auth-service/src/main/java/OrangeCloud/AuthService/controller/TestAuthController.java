package OrangeCloud.AuthService.controller;

import OrangeCloud.AuthService.dto.AuthResponse;
import OrangeCloud.AuthService.util.JwtTokenProvider;
import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.tags.Tag;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.Map;
import java.util.UUID;

/**
 * 부하 테스트용 토큰 발급 컨트롤러
 *
 * ENABLE_TEST_AUTH=true 환경변수가 설정된 경우에만 활성화됩니다.
 * 프로덕션에서는 부하 테스트 후 반드시 비활성화해야 합니다.
 */
@RestController
@RequestMapping("/api/test")
@Tag(name = "Test Authentication", description = "부하 테스트용 토큰 발급 API (테스트 환경 전용)")
@RequiredArgsConstructor
@Slf4j
@ConditionalOnProperty(name = "auth.test.enabled", havingValue = "true")
public class TestAuthController {

    private final JwtTokenProvider jwtTokenProvider;

    @Value("${auth.test.user-id:00000000-0000-0000-0000-000000000001}")
    private String testUserId;

    @Value("${auth.test.email:loadtest@wealist.co.kr}")
    private String testEmail;

    /**
     * 부하 테스트용 JWT 토큰 발급
     *
     * @return AccessToken, RefreshToken, UserId
     */
    @PostMapping("/token")
    @Operation(summary = "테스트 토큰 발급", description = "부하 테스트용 JWT 토큰을 발급합니다. 테스트 환경에서만 사용하세요.")
    public ResponseEntity<AuthResponse> generateTestToken(@RequestBody(required = false) Map<String, String> request) {
        log.warn("=== TEST TOKEN GENERATED === This endpoint should be disabled in production!");

        UUID userId;
        String email;

        // 요청에서 userId, email 추출 (없으면 기본값 사용)
        if (request != null && request.containsKey("userId")) {
            try {
                userId = UUID.fromString(request.get("userId"));
            } catch (IllegalArgumentException e) {
                userId = UUID.fromString(testUserId);
            }
        } else {
            userId = UUID.fromString(testUserId);
        }

        email = (request != null && request.containsKey("email"))
                ? request.get("email")
                : testEmail;

        String accessToken = jwtTokenProvider.generateToken(userId, email);
        String refreshToken = jwtTokenProvider.generateRefreshToken(userId, email);

        log.info("Test token generated for userId: {}, email: {}", userId, email);

        return ResponseEntity.ok(new AuthResponse(accessToken, refreshToken, userId));
    }

    /**
     * 테스트 엔드포인트 활성화 상태 확인
     */
    @GetMapping("/status")
    @Operation(summary = "테스트 모드 상태", description = "테스트 인증 엔드포인트 활성화 상태를 확인합니다.")
    public ResponseEntity<Map<String, Object>> getStatus() {
        return ResponseEntity.ok(Map.of(
            "testAuthEnabled", true,
            "message", "WARNING: Test auth endpoint is enabled. Disable after testing!"
        ));
    }
}
