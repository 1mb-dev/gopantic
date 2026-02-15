#!/bin/bash

# Generate Go documentation for gopantic package

echo "gopantic - API Documentation"
echo "=============================="

echo ""
echo "Package Overview:"
go doc ./pkg/model

echo ""
echo "Core Functions:"
go doc ./pkg/model ParseInto
go doc ./pkg/model ParseIntoWithFormat

echo ""
echo "Caching:"
go doc ./pkg/model NewCachedParser

echo ""
echo "Validation:"
go doc ./pkg/model ValidationError

echo ""
echo "Format Detection:"
go doc ./pkg/model DetectFormat

echo ""
echo "Usage:"
echo "  go doc -all ./pkg/model              # Full package docs"
echo "  go doc ./pkg/model <FunctionName>    # Specific function"
echo "  go doc -http :6060                   # Start HTML docs server"
echo ""

# Optional: Start HTTP documentation server
read -p "Start HTML documentation server? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "Starting documentation server on http://localhost:6060"
    echo "Visit: http://localhost:6060/pkg/github.com/1mb-dev/gopantic/pkg/model/"
    go doc -http :6060
fi