# Demonstração dos critérios de aceitação da Entrega 4 (Windows PowerShell 5.1+).
#
#   docker compose up -d --build
#   powershell -ExecutionPolicy Bypass -File scripts\entrega4-demo.ps1
#
# Parte 1 (segurança): rota protegida sem token -> 401; login no Authorization
# Server; token com "role" adulterada de "user" para "admin" -> 401.
# Parte 2 (escalabilidade): 9 requisições pelo Nginx distribuídas entre as 3
# réplicas do price-back; uma réplica é derrubada e as requisições continuam 200.

param(
    [string]$Gateway = "http://localhost",
    [string]$Username = "usuario-demo",
    [string]$Password = "SenhaDemo123!"
)

$ErrorActionPreference = "Stop"
$ProtectedRoute = "$Gateway/api/catalog/suggestions"   # price-back, exige papel admin

function Invoke-Api {
    param([string]$Method = "GET", [string]$Url, [string]$Token, [string]$Body)
    $headers = @{}
    if ($Token) { $headers["Authorization"] = "Bearer $Token" }
    $params = @{ Method = $Method; Uri = $Url; Headers = $headers; UseBasicParsing = $true }
    if ($Body) { $params["Body"] = $Body; $params["ContentType"] = "application/json" }
    try {
        $r = Invoke-WebRequest @params
        return [pscustomobject]@{ Status = [int]$r.StatusCode; Body = $r.Content; Headers = $r.Headers }
    } catch [System.Net.WebException] {
        $resp = $_.Exception.Response
        if (-not $resp) { throw }
        $reader = New-Object System.IO.StreamReader($resp.GetResponseStream())
        return [pscustomobject]@{ Status = [int]$resp.StatusCode; Body = $reader.ReadToEnd(); Headers = $resp.Headers }
    }
}

function ConvertFrom-Base64Url([string]$s) {
    $s = $s.Replace('-', '+').Replace('_', '/')
    switch ($s.Length % 4) { 2 { $s += '==' } 3 { $s += '=' } }
    [Text.Encoding]::UTF8.GetString([Convert]::FromBase64String($s))
}

function ConvertTo-Base64Url([string]$s) {
    [Convert]::ToBase64String([Text.Encoding]::UTF8.GetBytes($s)).TrimEnd('=').Replace('+', '-').Replace('/', '_')
}

Write-Host "`n=== PARTE 1 - SEGURANCA (JWT) ===" -ForegroundColor Cyan

Write-Host "`n[1] Rota protegida SEM token: GET /api/catalog/suggestions"
$r = Invoke-Api -Url $ProtectedRoute
Write-Host "    -> $($r.Status) $($r.Body)"

Write-Host "`n[2] Login no Authorization Server (POST /api/auth/login)"
$cred = @{ username = $Username; password = $Password } | ConvertTo-Json
$null = Invoke-Api -Method POST -Url "$Gateway/api/auth/signup" -Body $cred   # cria a conta na 1a vez (409 se ja existe)
$login = Invoke-Api -Method POST -Url "$Gateway/api/auth/login" -Body $cred
if ($login.Status -ne 200) { throw "login falhou: $($login.Status) $($login.Body)" }
$token = ($login.Body | ConvertFrom-Json).access_token
$parts = $token.Split('.')
Write-Host "    -> 200, JWT recebido"
Write-Host "    header : $(ConvertFrom-Base64Url $parts[0])"
Write-Host "    payload: $(ConvertFrom-Base64Url $parts[1])"

Write-Host "`n[3] Mesma rota com o token VERDADEIRO (papel user)"
$r = Invoke-Api -Url $ProtectedRoute -Token $token
Write-Host "    -> $($r.Status) $($r.Body)   (assinatura valida, mas papel sem permissao)"

Write-Host "`n[4] Token ADULTERADO: payload com role 'user' -> 'admin', assinatura original"
$payload = (ConvertFrom-Base64Url $parts[1]) -replace '"role":"user"', '"role":"admin"'
$forged = "$($parts[0]).$(ConvertTo-Base64Url $payload).$($parts[2])"
Write-Host "    payload adulterado: $payload"
$r = Invoke-Api -Url $ProtectedRoute -Token $forged
Write-Host "    -> $($r.Status) $($r.Body)   (o HMAC nao confere: requisicao recusada)"

Write-Host "`n=== PARTE 2 - ESCALABILIDADE (Nginx + 3 replicas) ===" -ForegroundColor Cyan

function Show-Requests([int]$n) {
    for ($i = 1; $i -le $n; $i++) {
        $r = Invoke-Api -Url "$Gateway/api/markets"
        Write-Host ("    req {0,2}: {1}  instancia={2}  upstream={3}" -f $i, $r.Status, $r.Headers["X-Instance"], $r.Headers["X-Upstream"])
    }
}

Write-Host "`n[5] 9 requisicoes seguidas para o Nginx (GET /api/markets)"
Show-Requests 9

$replicas = docker compose ps -q price-back
Write-Host "`n    replicas do price-back: $(@($replicas).Count)"
$victim = docker inspect --format "{{.Name}}" @($replicas)[0]
$victim = $victim.TrimStart('/')
Write-Host "`n[6] Derrubando uma replica: docker stop $victim"
docker stop $victim | Out-Null

Write-Host "`n[7] Novas requisicoes com uma replica fora do ar"
Show-Requests 6

Write-Host "`n[8] Religando a replica: docker start $victim"
docker start $victim | Out-Null

Write-Host "`nLogs do gateway (qual replica atendeu cada requisicao):" -ForegroundColor Cyan
docker compose logs --tail 20 gateway
