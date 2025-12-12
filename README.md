# TLS 인증서 폴더

이 폴더에 mkcert로 생성한 인증서 파일을 넣으면 `0.setup-cluster.sh` 스크립트가 자동으로 K8s Secret을 생성합니다.

## 인증서 생성 방법

```bash
# mkcert 설치 (Ubuntu/WSL)
sudo apt install libnss3-tools
curl -JLO "https://dl.filippo.io/mkcert/latest?for=linux/amd64"
chmod +x mkcert-v*-linux-amd64
sudo mv mkcert-v*-linux-amd64 /usr/local/bin/mkcert

# 로컬 CA 설치
mkcert -install

# 인증서 생성 (IP와 도메인 포함)
cd docker/scripts/dev/certs/
mkcert 192.168.0.3 local.wealist.co.kr localhost 127.0.0.1
```

## 생성되는 파일

- `192.168.0.3+3.pem` - 인증서
- `192.168.0.3+3-key.pem` - 개인키

이 파일들은 `.gitignore`에 의해 git에 커밋되지 않습니다.
