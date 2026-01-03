package OrangeCloud.AuthService.oauth;

import jakarta.servlet.ServletException;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import lombok.extern.slf4j.Slf4j;
import org.springframework.security.core.AuthenticationException;
import org.springframework.security.oauth2.core.OAuth2AuthenticationException;
import org.springframework.security.web.authentication.SimpleUrlAuthenticationFailureHandler;
import org.springframework.stereotype.Component;

import java.io.IOException;

/**
 * OAuth2 authentication failure handler for detailed error logging.
 */
@Component
@Slf4j
public class OAuth2FailureHandler extends SimpleUrlAuthenticationFailureHandler {

    public OAuth2FailureHandler() {
        setDefaultFailureUrl("/login?error");
    }

    @Override
    public void onAuthenticationFailure(HttpServletRequest request, HttpServletResponse response,
                                        AuthenticationException exception) throws IOException, ServletException {
        // Log detailed error information
        log.error("OAuth2 authentication failed: {}", exception.getMessage());

        if (exception instanceof OAuth2AuthenticationException oauth2Exception) {
            log.error("OAuth2 error code: {}", oauth2Exception.getError().getErrorCode());
            log.error("OAuth2 error description: {}", oauth2Exception.getError().getDescription());
            log.error("OAuth2 error URI: {}", oauth2Exception.getError().getUri());
        }

        // Log the full stack trace for debugging
        log.error("Full exception details:", exception);

        // Log request information for debugging
        log.error("Request URI: {}", request.getRequestURI());
        log.error("Request URL: {}", request.getRequestURL());
        log.error("Request query string: {}", request.getQueryString());
        log.error("Session ID: {}", request.getSession(false) != null ? request.getSession(false).getId() : "no session");

        // Log forwarded headers
        log.error("X-Forwarded-Proto: {}", request.getHeader("X-Forwarded-Proto"));
        log.error("X-Forwarded-Host: {}", request.getHeader("X-Forwarded-Host"));
        log.error("X-Forwarded-For: {}", request.getHeader("X-Forwarded-For"));
        log.error("Host header: {}", request.getHeader("Host"));

        super.onAuthenticationFailure(request, response, exception);
    }
}
