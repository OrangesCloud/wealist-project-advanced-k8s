package OrangeCloud.AuthService.config;

import OrangeCloud.AuthService.oauth.CustomOAuth2UserService;
import OrangeCloud.AuthService.oauth.OAuth2FailureHandler;
import OrangeCloud.AuthService.oauth.OAuth2RedirectUriFilter;
import OrangeCloud.AuthService.oauth.OAuth2SuccessHandler;
import lombok.RequiredArgsConstructor;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.security.config.annotation.web.builders.HttpSecurity;
import org.springframework.security.config.annotation.web.configuration.EnableWebSecurity;
import org.springframework.security.config.annotation.web.configurers.AbstractHttpConfigurer;
import org.springframework.security.config.http.SessionCreationPolicy;
import org.springframework.security.web.SecurityFilterChain;
import org.springframework.security.oauth2.client.web.OAuth2AuthorizationRequestRedirectFilter;
import org.springframework.web.cors.CorsConfiguration;
import org.springframework.web.cors.CorsConfigurationSource;
import org.springframework.web.cors.UrlBasedCorsConfigurationSource;
import org.springframework.web.filter.ForwardedHeaderFilter;

import java.util.Arrays;
import java.util.List;

@Configuration
@EnableWebSecurity
@RequiredArgsConstructor
public class SecurityConfig {

    private final CustomOAuth2UserService customOAuth2UserService;
    private final OAuth2SuccessHandler oAuth2SuccessHandler;
    private final OAuth2FailureHandler oAuth2FailureHandler;
    private final OAuth2RedirectUriFilter oAuth2RedirectUriFilter;

    @Bean
    public SecurityFilterChain securityFilterChain(HttpSecurity http) throws Exception {
        http
                // Must run BEFORE OAuth2AuthorizationRequestRedirectFilter to capture redirect_uri
                // before it redirects to the OAuth provider
                .addFilterBefore(oAuth2RedirectUriFilter, OAuth2AuthorizationRequestRedirectFilter.class)
                .cors(cors -> cors.configurationSource(corsConfigurationSource()))
                .csrf(AbstractHttpConfigurer::disable)
                // OAuth2 로그인은 세션이 필요함 (state 파라미터 저장용)
                // IF_REQUIRED: OAuth2 플로우 중에만 세션 생성, API 호출은 stateless
                .sessionManagement(session -> session.sessionCreationPolicy(SessionCreationPolicy.IF_REQUIRED))
                .authorizeHttpRequests(auth -> auth
                        // 공개 엔드포인트
                        .requestMatchers(
                                "/api/auth/**",
                                "/oauth2/**",
                                "/login/oauth2/**",
                                "/actuator/**",
                                "/health",
                                "/ready",
                                "/.well-known/**",  // JWKS 엔드포인트
                                "/swagger-ui/**",
                                "/v3/api-docs/**",
                                "/swagger-resources/**"
                        ).permitAll()
                        .anyRequest().authenticated()
                )
                .oauth2Login(oauth2 -> oauth2
                        .redirectionEndpoint(endpoint -> endpoint
                                .baseUri("/oauth2/callback/*")
                        )
                        .userInfoEndpoint(userInfo -> userInfo
                                .userService(customOAuth2UserService)
                        )
                        .successHandler(oAuth2SuccessHandler)
                        .failureHandler(oAuth2FailureHandler)
                );

        return http.build();
    }

    @Bean
    public CorsConfigurationSource corsConfigurationSource() {
        CorsConfiguration configuration = new CorsConfiguration();
        // allowCredentials: true 사용 시 origin에 "*" 사용 불가
        // allowedOriginPatterns로 패턴 매칭 사용
        configuration.setAllowedOriginPatterns(List.of(
            "http://localhost:*",           // 로컬 개발
            "https://wealist.co.kr",        // Production
            "https://www.wealist.co.kr",    // Production (www)
            "https://dev.wealist.co.kr",    // Dev 환경
            "https://*.wealist.co.kr"       // 서브도메인 전체
        ));
        configuration.setAllowedMethods(Arrays.asList("GET", "POST", "PUT", "DELETE", "OPTIONS"));
        configuration.setAllowedHeaders(List.of("*"));
        configuration.setAllowCredentials(true);  // credentials 허용 (axios withCredentials: true)

        UrlBasedCorsConfigurationSource source = new UrlBasedCorsConfigurationSource();
        source.registerCorsConfiguration("/**", configuration);
        return source;
    }

    /**
     * X-Forwarded-* 헤더를 처리하여 프록시 뒤에서 HTTPS 스킴을 올바르게 인식
     * CloudFront/ALB 뒤에서 OAuth2 리다이렉트 URL이 HTTP로 생성되는 문제 해결
     */
    @Bean
    public ForwardedHeaderFilter forwardedHeaderFilter() {
        return new ForwardedHeaderFilter();
    }
}
