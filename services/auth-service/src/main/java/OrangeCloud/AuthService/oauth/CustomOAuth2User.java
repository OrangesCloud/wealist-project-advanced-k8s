package OrangeCloud.AuthService.oauth;

import lombok.Getter;
import org.springframework.security.core.GrantedAuthority;
import org.springframework.security.core.authority.SimpleGrantedAuthority;
import org.springframework.security.oauth2.core.user.OAuth2User;

import java.io.Serializable;
import java.util.Collection;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.UUID;
import java.util.stream.Collectors;

/**
 * CustomOAuth2User - OAuth2 인증 사용자 정보를 담는 클래스
 *
 * Spring Session Redis를 사용하므로 Serializable 구현 필수.
 * OAuth2User는 직렬화가 안 되므로, 필요한 데이터만 직렬화 가능한 형태로 저장.
 */
@Getter
public class CustomOAuth2User implements OAuth2User, Serializable {

    private static final long serialVersionUID = 1L;

    // OAuth2User의 attributes를 직렬화 가능한 HashMap으로 저장
    private final Map<String, Object> attributes;

    // authorities를 직렬화 가능한 형태로 저장 (권한 문자열 리스트)
    private final List<String> authorityStrings;

    private final UUID userId;
    private final String email;
    private final String name;

    public CustomOAuth2User(OAuth2User oAuth2User, UUID userId, String email, String name) {
        // attributes를 직렬화 가능한 HashMap으로 복사
        this.attributes = new HashMap<>(oAuth2User.getAttributes());

        // authorities를 문자열 리스트로 변환하여 저장
        this.authorityStrings = oAuth2User.getAuthorities().stream()
                .map(GrantedAuthority::getAuthority)
                .collect(Collectors.toList());

        this.userId = userId;
        this.email = email;
        this.name = name;
    }

    @Override
    public Map<String, Object> getAttributes() {
        return attributes;
    }

    @Override
    public Collection<? extends GrantedAuthority> getAuthorities() {
        // 문자열 리스트를 SimpleGrantedAuthority로 변환하여 반환
        return authorityStrings.stream()
                .map(SimpleGrantedAuthority::new)
                .collect(Collectors.toList());
    }

    @Override
    public String getName() {
        return name;
    }
}
