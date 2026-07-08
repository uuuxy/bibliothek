#!/bin/bash
# Since docker is failing due to some weird overlay issue on the environment, we'll test locally by simply re-running trivy on our local project tree instead of inside docker.
~/bin/trivy fs --severity HIGH,CRITICAL --ignore-unfixed .
