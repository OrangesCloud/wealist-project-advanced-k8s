#!/bin/bash

# Istio 설정 및 상태 확인 스크립트

set -e

# 색상 정의
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
NC='\033[0m' # No Color

# 함수 정의
log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
}

log_header() {
    echo -e "${PURPLE}🔍 $1${NC}"
    echo "=================================================="
}

# 도움말
show_help() {
    echo "Istio 설정 및 상태 확인 스크립트"
    echo ""
    echo "사용법: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  --namespace, -n NAMESPACE    대상 네임스페이스 (기본: wealist-dev)"
    echo "  --verbose, -v                상세 출력"
    echo "  --help, -h                   도움말 표시"
    echo ""
    echo "예시:"
    echo "  $0                           # dev 환경 확인"
    echo "  $0 -n wealist-prod           # prod 환경 확인"
    echo "  $0 -v                        # 상세 출력"
}

# 기본값
NAMESPACE="wealist-prod"
VERBOSE=false

# 파라미터 파싱
while [[ $# -gt 0 ]]; do
    case $1 in
        -n|--namespace)
            NAMESPACE="$2"
            shift 2
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -h|--help)
            show_help
            exit 0
            ;;
        *)
            log_error "알 수 없는 옵션: $1"
            show_help
            exit 1
            ;;
    esac
done

log_info "Istio 상태 확인 시작 (네임스페이스: $NAMESPACE)"
echo ""

# =============================================================================
# 1. Istio 설치 상태 확인
# =============================================================================
log_header "1. Istio 설치 상태 확인"

# Istio 네임스페이스 확인
if kubectl get namespace istio-system >/dev/null 2>&1; then
    log_success "istio-system 네임스페이스 존재"
else
    log_error "istio-system 네임스페이스가 존재하지 않습니다"
    log_info "Istio가 설치되지 않았을 수 있습니다"
    exit 1
fi

# Istio 컨트롤 플레인 확인
log_info "Istio 컨트롤 플레인 상태:"
if kubectl get pods -n istio-system --no-headers 2>/dev/null | while read line; do
    pod_name=$(echo $line | awk '{print $1}')
    pod_status=$(echo $line | awk '{print $3}')
    if [ "$pod_status" = "Running" ]; then
        log_success "  $pod_name: $pod_status"
    else
        log_warning "  $pod_name: $pod_status"
    fi
done; then
    :
else
    log_error "Istio 컨트롤 플레인 Pod 조회 실패"
fi

# Istio 버전 확인
if command -v istioctl &> /dev/null; then
    ISTIO_VERSION=$(istioctl version --short 2>/dev/null || echo "확인 불가")
    log_info "Istio 버전: $ISTIO_VERSION"
else
    log_warning "istioctl CLI가 설치되지 않았습니다"
fi

echo ""

# =============================================================================
# 2. Gateway 및 HTTPRoute 확인
# =============================================================================
log_header "2. Gateway 및 HTTPRoute 확인"

# Gateway 확인
log_info "Gateway 상태:"
if kubectl get gateway -n istio-system --no-headers 2>/dev/null | while read line; do
    gateway_name=$(echo $line | awk '{print $1}')
    log_success "  Gateway: $gateway_name"
    if [ "$VERBOSE" = true ]; then
        kubectl describe gateway $gateway_name -n istio-system | grep -A 5 "Spec:"
    fi
done; then
    :
else
    log_warning "Gateway가 없거나 조회할 수 없습니다"
fi

# HTTPRoute 확인
log_info "HTTPRoute 상태:"
if kubectl get httproute -n $NAMESPACE --no-headers 2>/dev/null | while read line; do
    route_name=$(echo $line | awk '{print $1}')
    log_success "  HTTPRoute: $route_name"
    if [ "$VERBOSE" = true ]; then
        kubectl describe httproute $route_name -n $NAMESPACE | grep -A 10 "Spec:"
    fi
done; then
    :
else
    log_warning "HTTPRoute가 없거나 조회할 수 없습니다"
fi

echo ""

# =============================================================================
# 3. 서비스 메시 상태 확인
# =============================================================================
log_header "3. 서비스 메시 상태 확인"

# 네임스페이스 라벨 확인
log_info "네임스페이스 Istio 라벨:"
NAMESPACE_LABELS=$(kubectl get namespace $NAMESPACE -o jsonpath='{.metadata.labels}' 2>/dev/null || echo "{}")
if echo "$NAMESPACE_LABELS" | grep -q "istio"; then
    log_success "  Istio 라벨이 설정되어 있습니다"
    if [ "$VERBOSE" = true ]; then
        echo "  라벨: $NAMESPACE_LABELS"
    fi
else
    log_warning "  Istio 라벨이 설정되지 않았습니다"
    log_info "  다음 명령으로 설정하세요: kubectl label namespace $NAMESPACE istio-injection=enabled"
fi

# Pod의 사이드카 상태 확인
log_info "Pod 사이드카 상태:"
if kubectl get pods -n $NAMESPACE --no-headers 2>/dev/null | while read line; do
    pod_name=$(echo $line | awk '{print $1}')
    containers=$(echo $line | awk '{print $2}')
    
    # 컨테이너 수가 2개 이상이면 사이드카가 있을 가능성
    if [[ "$containers" == *"/"* ]]; then
        container_count=$(echo $containers | cut -d'/' -f2)
        if [ "$container_count" -gt 1 ]; then
            log_success "  $pod_name: 사이드카 있음 ($containers)"
        else
            log_warning "  $pod_name: 사이드카 없음 ($containers)"
        fi
    fi
done; then
    :
else
    log_warning "Pod 조회 실패 또는 Pod가 없습니다"
fi

echo ""

# =============================================================================
# 4. mTLS 상태 확인
# =============================================================================
log_header "4. mTLS 상태 확인"

# PeerAuthentication 확인
log_info "PeerAuthentication 정책:"
if kubectl get peerauthentication -n $NAMESPACE --no-headers 2>/dev/null | while read line; do
    policy_name=$(echo $line | awk '{print $1}')
    log_success "  PeerAuthentication: $policy_name"
    if [ "$VERBOSE" = true ]; then
        kubectl get peerauthentication $policy_name -n $NAMESPACE -o yaml | grep -A 5 "spec:"
    fi
done; then
    :
else
    log_info "  PeerAuthentication 정책이 없습니다 (기본 설정 사용)"
fi

# DestinationRule 확인
log_info "DestinationRule 상태:"
if kubectl get destinationrule -n $NAMESPACE --no-headers 2>/dev/null | while read line; do
    dr_name=$(echo $line | awk '{print $1}')
    log_success "  DestinationRule: $dr_name"
    if [ "$VERBOSE" = true ]; then
        kubectl describe destinationrule $dr_name -n $NAMESPACE | grep -A 10 "Traffic Policy:"
    fi
done; then
    :
else
    log_warning "DestinationRule이 없습니다"
fi

echo ""

# =============================================================================
# 5. AuthorizationPolicy 확인
# =============================================================================
log_header "5. AuthorizationPolicy 확인"

log_info "AuthorizationPolicy 상태:"
if kubectl get authorizationpolicy -n $NAMESPACE --no-headers 2>/dev/null | while read line; do
    policy_name=$(echo $line | awk '{print $1}')
    log_success "  AuthorizationPolicy: $policy_name"
    if [ "$VERBOSE" = true ]; then
        kubectl describe authorizationpolicy $policy_name -n $NAMESPACE | grep -A 10 "Spec:"
    fi
done; then
    :
else
    log_info "  AuthorizationPolicy가 없습니다 (모든 트래픽 허용)"
fi

echo ""

# =============================================================================
# 6. Ambient 모드 확인 (해당하는 경우)
# =============================================================================
log_header "6. Ambient 모드 확인"

# Waypoint 확인
log_info "Waypoint 프록시 상태:"
if kubectl get pods -n $NAMESPACE -l gateway.istio.io/managed=istio.io-waypoint-controller --no-headers 2>/dev/null | while read line; do
    waypoint_name=$(echo $line | awk '{print $1}')
    waypoint_status=$(echo $line | awk '{print $3}')
    if [ "$waypoint_status" = "Running" ]; then
        log_success "  Waypoint: $waypoint_name ($waypoint_status)"
    else
        log_warning "  Waypoint: $waypoint_name ($waypoint_status)"
    fi
done; then
    :
else
    log_info "  Waypoint 프록시가 없습니다 (Sidecar 모드 또는 Ambient 비활성화)"
fi

# ztunnel 확인
log_info "ztunnel (L4 프록시) 상태:"
if kubectl get pods -n istio-system -l app=ztunnel --no-headers 2>/dev/null | while read line; do
    ztunnel_name=$(echo $line | awk '{print $1}')
    ztunnel_status=$(echo $line | awk '{print $3}')
    if [ "$ztunnel_status" = "Running" ]; then
        log_success "  ztunnel: $ztunnel_name ($ztunnel_status)"
    else
        log_warning "  ztunnel: $ztunnel_name ($ztunnel_status)"
    fi
done; then
    :
else
    log_info "  ztunnel이 없습니다 (Ambient 모드 비활성화)"
fi

echo ""

# =============================================================================
# 7. 텔레메트리 확인
# =============================================================================
log_header "7. 텔레메트리 확인"

# Telemetry 리소스 확인
log_info "Telemetry 설정:"
if kubectl get telemetry -n $NAMESPACE --no-headers 2>/dev/null | while read line; do
    telemetry_name=$(echo $line | awk '{print $1}')
    log_success "  Telemetry: $telemetry_name"
done; then
    :
else
    log_info "  Telemetry 설정이 없습니다 (기본 설정 사용)"
fi

# Kiali 확인
log_info "Kiali 상태:"
if kubectl get pods -n istio-system -l app=kiali --no-headers 2>/dev/null | while read line; do
    kiali_name=$(echo $line | awk '{print $1}')
    kiali_status=$(echo $line | awk '{print $3}')
    if [ "$kiali_status" = "Running" ]; then
        log_success "  Kiali: $kiali_name ($kiali_status)"
    else
        log_warning "  Kiali: $kiali_name ($kiali_status)"
    fi
done; then
    :
else
    log_warning "  Kiali가 설치되지 않았습니다"
fi

# Jaeger 확인
log_info "Jaeger 상태:"
if kubectl get pods -n istio-system -l app=jaeger --no-headers 2>/dev/null | while read line; do
    jaeger_name=$(echo $line | awk '{print $1}')
    jaeger_status=$(echo $line | awk '{print $3}')
    if [ "$jaeger_status" = "Running" ]; then
        log_success "  Jaeger: $jaeger_name ($jaeger_status)"
    else
        log_warning "  Jaeger: $jaeger_name ($jaeger_status)"
    fi
done; then
    :
else
    log_warning "  Jaeger가 설치되지 않았습니다"
fi

echo ""

# =============================================================================
# 8. 연결성 테스트 (선택사항)
# =============================================================================
log_header "8. 연결성 테스트"

log_info "서비스 연결성 확인:"
SERVICES=("auth-service" "board-service" "user-service" "chat-service" "noti-service" "storage-service" "video-service")

for service in "${SERVICES[@]}"; do
    if kubectl get service $service -n $NAMESPACE >/dev/null 2>&1; then
        # 서비스 엔드포인트 확인
        ENDPOINTS=$(kubectl get endpoints $service -n $NAMESPACE -o jsonpath='{.subsets[*].addresses[*].ip}' 2>/dev/null || echo "")
        if [ -n "$ENDPOINTS" ]; then
            log_success "  $service: 엔드포인트 있음"
        else
            log_warning "  $service: 엔드포인트 없음 (Pod가 Ready 상태가 아님)"
        fi
    else
        log_info "  $service: 서비스 없음"
    fi
done

echo ""

# =============================================================================
# 9. 요약 및 권장사항
# =============================================================================
log_header "9. 요약 및 권장사항"

log_info "Istio 상태 확인 완료!"
echo ""
log_info "추가 확인 명령어:"
echo "  # Istio 설정 검증"
echo "  istioctl analyze -n $NAMESPACE"
echo ""
echo "  # 프록시 상태 확인"
echo "  istioctl proxy-status"
echo ""
echo "  # 서비스 메시 시각화 (Kiali)"
echo "  kubectl port-forward -n istio-system svc/kiali 20001:20001"
echo "  # 브라우저에서 http://localhost:20001"
echo ""
echo "  # 분산 추적 (Jaeger)"
echo "  kubectl port-forward -n istio-system svc/tracing 16686:80"
echo "  # 브라우저에서 http://localhost:16686"
echo ""
echo "  # mTLS 상태 확인"
echo "  istioctl authn tls-check $service.$NAMESPACE.svc.cluster.local"

log_success "Istio 상태 확인이 완료되었습니다!"