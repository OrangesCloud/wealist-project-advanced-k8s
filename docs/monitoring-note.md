make kind-localhost-setup
make helm-install-all ENV=localhost

make kind-dev-setup
make helm-install-all ENV=dev
