// Runtime configuration (injected by K8s ConfigMap or CI/CD)
// This file is loaded before the main app bundle
//
// Environments:
//   - local (docker-compose): Set API_BASE_URL = "http://localhost"
//   - production (CloudFront + K8s): Set API_BASE_URL = "" (empty = relative paths)
//
// WebSocket/SSE connections bypass CloudFront and connect directly to API_DOMAIN
// to avoid CloudFront HTTP/2 and WebSocket protocol issues

window.__ENV__ = {
  // Empty string = use relative paths (CloudFront/ingress mode)
  // The frontend will use /svc/* paths which CloudFront routes to backend
  API_BASE_URL: "",
  // Direct API domain for WebSocket/SSE (bypasses CloudFront)
  // Set this to your API origin domain (e.g., "api.dev.wealist.co.kr")
  API_DOMAIN: ""
};
