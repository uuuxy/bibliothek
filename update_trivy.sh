#!/bin/bash
sed -i 's/uses: aquasecurity\/trivy-action@master/uses: aquasecurity\/trivy-action@master\n        env:\n          TRIVY_OFFLINE_SCAN: "true"/' .github/workflows/security-scan.yml
