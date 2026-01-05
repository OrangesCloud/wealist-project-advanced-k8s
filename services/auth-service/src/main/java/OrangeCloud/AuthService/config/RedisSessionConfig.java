package OrangeCloud.AuthService.config;

import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.context.annotation.Configuration;
import org.springframework.data.redis.connection.RedisConnectionFactory;
import org.springframework.session.data.redis.config.annotation.web.http.EnableRedisHttpSession;
import org.springframework.session.web.context.AbstractHttpSessionApplicationInitializer;

import jakarta.annotation.PostConstruct;

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
 *
 * NOTE: redisNamespace를 설정하여 키 prefix를 wealist:auth:session으로 함
 */
@Configuration
@EnableRedisHttpSession(
    maxInactiveIntervalInSeconds = 1800, // 30분
    redisNamespace = "wealist:auth:session"
)
@Slf4j
public class RedisSessionConfig extends AbstractHttpSessionApplicationInitializer {

    @Autowired
    private RedisConnectionFactory redisConnectionFactory;

    @PostConstruct
    public void logRedisConnection() {
        try {
            // Redis 연결 테스트
            var connection = redisConnectionFactory.getConnection();
            log.info("✅ Redis Session connected successfully. Redis info: {}",
                connection.serverCommands().info("server").substring(0, 100));
            connection.close();
        } catch (Exception e) {
            log.error("❌ Redis Session connection FAILED: {}", e.getMessage(), e);
        }
    }
}
