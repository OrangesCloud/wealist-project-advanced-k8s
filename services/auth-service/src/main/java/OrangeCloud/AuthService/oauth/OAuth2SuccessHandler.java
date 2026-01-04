package OrangeCloud.AuthService.oauth;

import OrangeCloud.AuthService.dto.AuthResponse;
import OrangeCloud.AuthService.service.AuthService;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import jakarta.servlet.http.HttpSession;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.security.core.Authentication;
import org.springframework.security.web.authentication.SimpleUrlAuthenticationSuccessHandler;
import org.springframework.stereotype.Component;
import org.springframework.web.util.UriComponentsBuilder;

import java.io.IOException;

@Component
@RequiredArgsConstructor
@Slf4j
public class OAuth2SuccessHandler extends SimpleUrlAuthenticationSuccessHandler {

    private final AuthService authService;

    @Value("${oauth2.redirect-url}")
    private String defaultRedirectUrl;

    @Override
    public void onAuthenticationSuccess(HttpServletRequest request, HttpServletResponse response,
                                        Authentication authentication) throws IOException {
        CustomOAuth2User oAuth2User = (CustomOAuth2User) authentication.getPrincipal();

        log.info("OAuth2 인증 성공: userId={}, email={}", oAuth2User.getUserId(), oAuth2User.getEmail());

        // 토큰 발행 (email claim 포함 - ops-portal 등에서 필요)
        AuthResponse authResponse = authService.generateTokens(oAuth2User.getUserId(), oAuth2User.getEmail());

        // 클라이언트가 지정한 redirect_uri 확인 (세션에서)
        String redirectUrl = getClientRedirectUri(request);

        // 프론트엔드로 리다이렉트 (토큰 정보를 쿼리 파라미터로 전달)
        // nickName, email은 더 이상 전달하지 않음 - 프론트에서 user-service 호출
        String targetUrl = UriComponentsBuilder.fromUriString(redirectUrl)
                .queryParam("accessToken", authResponse.getAccessToken())
                .queryParam("refreshToken", authResponse.getRefreshToken())
                .queryParam("userId", authResponse.getUserId().toString())
                .build()
                .toUriString();

        log.debug("Redirecting to: {}", targetUrl);

        // 세션에서 redirect_uri 제거 (cleanup)
        HttpSession session = request.getSession(false);
        if (session != null) {
            session.removeAttribute(OAuth2RedirectUriFilter.REDIRECT_URI_SESSION_KEY);
        }

        getRedirectStrategy().sendRedirect(request, response, targetUrl);
    }

    /**
     * Get the redirect URI from session (set by OAuth2RedirectUriFilter) or use default.
     */
    private String getClientRedirectUri(HttpServletRequest request) {
        HttpSession session = request.getSession(false);
        if (session != null) {
            String clientRedirectUri = (String) session.getAttribute(OAuth2RedirectUriFilter.REDIRECT_URI_SESSION_KEY);
            if (clientRedirectUri != null && !clientRedirectUri.isBlank()) {
                log.debug("Using client-specified redirect_uri: {}", clientRedirectUri);
                return clientRedirectUri;
            }
        }
        log.debug("Using default redirect_url: {}", defaultRedirectUrl);
        return defaultRedirectUrl;
    }
}
