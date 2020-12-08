param($network = "testnet")

$ErrorActionPreference = "Stop"

$networkDir = "$HOME\AppData\Local\XudDocker\mainnet"
$dataDir = "$networkDir\data"
$proxyDir = "$dataDir\proxy"
$uiBuild = "$HOME\xud-ui-dashboard\build"

Write-Output "proxyDir=$proxyDir"

docker build . -t proxy
if ($LASTEXITCODE -ne 0)
{
    exit 1
}
docker run -it --rm --name proxy `
-e "NETWORK=$network" `
-e "GIN_MODE=release" `
--net "${network}_default" `
-p 8080:8080 `
-v //var/run/docker.sock: /var/run/docker.sock `
-v "${proxyDir}:/root/.proxy" `
-v "${networkDir}:/root/network:ro" `
-v "${uiBuild}:/ui" `
proxy
