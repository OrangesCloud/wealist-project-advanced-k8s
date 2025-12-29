package OrangeCloud.AuthService.config;

import org.springframework.context.annotation.Configuration;
import org.springframework.session.data.redis.config.annotation.web.http.EnableRedisHttpSession;

/**
 * Redis 기반 HTTP 세션 설정.
 *
 * OAuth2 로그인 시 state 파라미터가 세션에 저장되는데,
 * 멀티 pod 환경에서 세션을 Redis에 저장하여 모든 pod가 공유할 수 있도록 함.
 *
 * 문제 상황:
 * - Pod A에서 OAuth2 로그인 시작 → state를 메모리 세션에 저장
 * - Google 콜백이 Pod B로 라우팅됨 → state를 찾을 수 없어 실패
 *
 * 해결:
 * - 세션을 Redis에 저장하여 모든 pod가 동일한 세션 접근 가능
 */
@Configuration
@EnableRedisHttpSession(maxInactiveIntervalInSeconds = 1800) // 30분
public class RedisSessionConfig {
    // Spring Session이 자동으로 Redis에 세션 저장
    // 기존 RedisConfig의 RedisConnectionFactory를 재사용
}
