// Runtime configuration (injected by K8s ConfigMap or CI/CD)
// This file is loaded before the main app bundle
//
// Environments:
//   - local (docker-compose): Set API_BASE_URL = "http://localhost"
//   - production (CloudFront + K8s): Set API_BASE_URL = "" (empty = relative paths)

window.__ENV__ = {
  // Empty string = use relative paths (CloudFront/ingress mode)
  // The frontend will use /svc/* paths which CloudFront routes to backend
  API_BASE_URL: '',
};
