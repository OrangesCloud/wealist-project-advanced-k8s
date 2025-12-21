package OrangeCloud.AuthService.config;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.security.*;
import java.security.interfaces.RSAPrivateKey;
import java.security.interfaces.RSAPublicKey;
import java.security.spec.PKCS8EncodedKeySpec;
import java.security.spec.X509EncodedKeySpec;
import java.util.Base64;

/**
 * RSA 키 쌍 설정
 * - 환경변수에서 PEM 형식 키를 로드하거나
 * - 키가 없으면 런타임에 생성 (개발용)
 */
@Configuration
public class RsaKeyConfig {

    private static final Logger logger = LoggerFactory.getLogger(RsaKeyConfig.class);

    @Value("${jwt.rsa.public-key:}")
    private String publicKeyPem;

    @Value("${jwt.rsa.private-key:}")
    private String privateKeyPem;

    @Value("${jwt.rsa.key-id:wealist-auth-key-1}")
    private String keyId;

    private KeyPair keyPair;

    @Bean
    public KeyPair rsaKeyPair() throws Exception {
        if (keyPair != null) {
            return keyPair;
        }

        // 환경변수에서 키 로드 시도
        if (publicKeyPem != null && !publicKeyPem.isEmpty()
            && privateKeyPem != null && !privateKeyPem.isEmpty()) {
            logger.info("Loading RSA key pair from environment variables");
            keyPair = loadKeyPairFromPem(publicKeyPem, privateKeyPem);
        } else {
            // 개발용: 런타임에 키 생성
            logger.warn("RSA keys not configured, generating new key pair (development only!)");
            keyPair = generateKeyPair();
        }

        return keyPair;
    }

    @Bean
    public RSAPublicKey rsaPublicKey(KeyPair keyPair) {
        return (RSAPublicKey) keyPair.getPublic();
    }

    @Bean
    public RSAPrivateKey rsaPrivateKey(KeyPair keyPair) {
        return (RSAPrivateKey) keyPair.getPrivate();
    }

    @Bean
    public String rsaKeyId() {
        return keyId;
    }

    private KeyPair generateKeyPair() throws NoSuchAlgorithmException {
        KeyPairGenerator generator = KeyPairGenerator.getInstance("RSA");
        generator.initialize(2048);
        KeyPair pair = generator.generateKeyPair();

        // 개발용으로 생성된 키 로깅 (프로덕션에서는 제거 필요)
        logger.debug("Generated RSA Public Key (for development):\n{}",
            formatKeyAsPem(pair.getPublic().getEncoded(), "PUBLIC KEY"));

        return pair;
    }

    private KeyPair loadKeyPairFromPem(String publicPem, String privatePem) throws Exception {
        RSAPublicKey publicKey = loadPublicKey(publicPem);
        RSAPrivateKey privateKey = loadPrivateKey(privatePem);
        return new KeyPair(publicKey, privateKey);
    }

    private RSAPublicKey loadPublicKey(String pem) throws Exception {
        String publicKeyContent = pem
            .replace("-----BEGIN PUBLIC KEY-----", "")
            .replace("-----END PUBLIC KEY-----", "")
            .replaceAll("\\s", "");

        byte[] decoded = Base64.getDecoder().decode(publicKeyContent);
        X509EncodedKeySpec spec = new X509EncodedKeySpec(decoded);
        KeyFactory factory = KeyFactory.getInstance("RSA");
        return (RSAPublicKey) factory.generatePublic(spec);
    }

    private RSAPrivateKey loadPrivateKey(String pem) throws Exception {
        String privateKeyContent = pem
            .replace("-----BEGIN PRIVATE KEY-----", "")
            .replace("-----END PRIVATE KEY-----", "")
            .replaceAll("\\s", "");

        byte[] decoded = Base64.getDecoder().decode(privateKeyContent);
        PKCS8EncodedKeySpec spec = new PKCS8EncodedKeySpec(decoded);
        KeyFactory factory = KeyFactory.getInstance("RSA");
        return (RSAPrivateKey) factory.generatePrivate(spec);
    }

    private String formatKeyAsPem(byte[] keyBytes, String type) {
        String base64 = Base64.getEncoder().encodeToString(keyBytes);
        StringBuilder sb = new StringBuilder();
        sb.append("-----BEGIN ").append(type).append("-----\n");
        for (int i = 0; i < base64.length(); i += 64) {
            sb.append(base64, i, Math.min(i + 64, base64.length())).append("\n");
        }
        sb.append("-----END ").append(type).append("-----");
        return sb.toString();
    }
}
