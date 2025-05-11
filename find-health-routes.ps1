Write-Host "Finding all occurrences of '/health' in the codebase..."
Get-ChildItem -Path . -Recurse -Include *.go | ForEach-Object {
    $file = $_
    $content = Get-Content $file.FullName -Raw
    if ($content -like '*"/health"*') {
        Write-Host "Found in: $($file.FullName)"
        $lineNumber = 1
        Get-Content $file.FullName | ForEach-Object {
            if ($_ -like '*"/health"*') {
                Write-Host "Line $lineNumber`: $_"
            }
            $lineNumber++
        }
        Write-Host "----------------------------"
    }
}

Write-Host "Finding all occurrences of 'health' in the codebase..."
Get-ChildItem -Path . -Recurse -Include *.go | ForEach-Object {
    $file = $_
    $content = Get-Content $file.FullName -Raw
    if ($content -like '*health*' -and $file.FullName -notlike '*find-health-routes.ps1*') {
        Write-Host "Found in: $($file.FullName)"
        $lineNumber = 1
        Get-Content $file.FullName | ForEach-Object {
            if ($_ -like '*health*') {
                Write-Host "Line $lineNumber`: $_"
            }
            $lineNumber++
        }
        Write-Host "----------------------------"
    }
} 