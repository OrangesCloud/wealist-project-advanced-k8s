package OrangeCloud.AuthService.config;

import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.session.web.http.CookieSerializer;
import org.springframework.session.web.http.DefaultCookieSerializer;

/**
 * Session Cookie Configuration for multi-pod OAuth2 flow.
 *
 * OAuth2 flow uses session to store authorization request (state parameter).
 * In multi-pod deployment with reverse proxy (CloudFront/nginx),
 * proper cookie configuration is essential for session to work across requests.
 */
@Configuration
public class SessionConfig {

    @Bean
    public CookieSerializer cookieSerializer() {
        DefaultCookieSerializer serializer = new DefaultCookieSerializer();

        // Cookie name (default is SESSION)
        serializer.setCookieName("SESSION");

        // Cookie path - must be / to work across all endpoints
        serializer.setCookiePath("/");

        // SameSite=Lax allows cookie to be sent on top-level navigation (OAuth redirect)
        // SameSite=None would require Secure=true
        serializer.setSameSite("Lax");

        // Use secure cookie in production (HTTPS)
        // This is determined by X-Forwarded-Proto header from reverse proxy
        serializer.setUseSecureCookie(true);

        // HttpOnly for security
        serializer.setUseHttpOnlyCookie(true);

        // Max age in seconds (30 minutes, same as @EnableRedisHttpSession)
        serializer.setCookieMaxAge(1800);

        return serializer;
    }
}
