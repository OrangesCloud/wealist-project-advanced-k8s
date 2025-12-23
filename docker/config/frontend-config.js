// Runtime configuration for Docker-compose (local development)
// This file overrides the default config.js in K8s mode

window.__ENV__ = {
  // Docker-compose: nginx gateway on port 80
  API_BASE_URL: "http://localhost",
  // Direct API domain (not used in docker-compose)
  API_DOMAIN: ""
};
