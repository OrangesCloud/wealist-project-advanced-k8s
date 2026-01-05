package OrangeCloud.AuthService.oauth;

import jakarta.servlet.FilterChain;
import jakarta.servlet.ServletException;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;
import org.springframework.util.AntPathMatcher;
import org.springframework.web.filter.OncePerRequestFilter;

import java.io.IOException;
import java.util.List;

/**
 * Filter to capture and store client's redirect_uri for OAuth2 flow.
 * This enables different frontends (main app, ops-portal) to use the same auth-service.
 */
@Component
@Slf4j
public class OAuth2RedirectUriFilter extends OncePerRequestFilter {

    public static final String REDIRECT_URI_SESSION_KEY = "oauth2_client_redirect_uri";

    @Value("${oauth2.allowed-redirect-patterns:}")
    private List<String> allowedRedirectPatterns;

    private final AntPathMatcher pathMatcher = new AntPathMatcher();

    @Override
    protected void doFilterInternal(HttpServletRequest request, HttpServletResponse response,
                                    FilterChain filterChain) throws ServletException, IOException {
        String requestUri = request.getRequestURI();

        // Check for OAuth2 authorization requests (with or without /api prefix)
        boolean isAuthorizationRequest = requestUri.startsWith("/oauth2/authorization/")
                || requestUri.startsWith("/api/oauth2/authorization/");

        if (isAuthorizationRequest) {
            log.info("OAuth2 authorization request detected: {}", requestUri);
            String clientRedirectUri = request.getParameter("redirect_uri");
            log.info("Client redirect_uri parameter: {}", clientRedirectUri);

            if (clientRedirectUri != null && !clientRedirectUri.isBlank()) {
                if (isAllowedRedirectUri(clientRedirectUri)) {
                    request.getSession().setAttribute(REDIRECT_URI_SESSION_KEY, clientRedirectUri);
                    log.info("Stored client redirect_uri in session: {}", clientRedirectUri);
                } else {
                    log.warn("Rejected invalid redirect_uri: {}", clientRedirectUri);
                }
            } else {
                log.info("No redirect_uri parameter provided, will use default");
            }
        }
        filterChain.doFilter(request, response);
    }

    private boolean isAllowedRedirectUri(String redirectUri) {
        // If no patterns configured, allow all (for backward compatibility)
        if (allowedRedirectPatterns == null || allowedRedirectPatterns.isEmpty()) {
            log.debug("No redirect patterns configured, allowing all");
            return true;
        }

        for (String pattern : allowedRedirectPatterns) {
            if (pattern.isBlank()) continue;
            // Support wildcard patterns like https://wealist.co.kr/**
            if (pathMatcher.match(pattern, redirectUri)) {
                return true;
            }
            // Also check if URI starts with pattern (for simple prefix matching)
            if (redirectUri.startsWith(pattern.replace("**", ""))) {
                return true;
            }
        }
        return false;
    }
}
