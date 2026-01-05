package OrangeCloud.AuthService.config;

import org.springframework.context.annotation.Configuration;
import org.springframework.session.data.redis.config.annotation.web.http.EnableRedisHttpSession;

/**
 * Redis 기반 HTTP 세션 설정.
 *
 * OAuth2 로그인 시 state 파라미터가 세션에 저장되는데,
 * 멀티 pod 환경에서 세션을 Redis에 저장하여 모든 pod가 공유할 수 있도록 함.
 */
@Configuration
@EnableRedisHttpSession(
    maxInactiveIntervalInSeconds = 1800,
    redisNamespace = "wealist:auth:session"
)
public class RedisSessionConfig {
    // Spring Session이 자동으로 Redis에 세션 저장
}
