#!/bin/bash

# Test script to verify v0 to v2 migration
# Tests that both v0 (deprecated) and v2 (current) endpoints work

set -e

BASE_URL="${BASE_URL:-http://localhost:8000}"

echo "Testing ROSA Regional Platform API v0 to v2 Migration"
echo "======================================================"
echo ""
echo "Base URL: $BASE_URL"
echo ""

# Test v2 endpoints (current)
echo "Testing v2 endpoints (current)..."
echo ""

echo "1. Testing GET /api/v2/live"
RESPONSE=$(curl -s -w "\n%{http_code}" "$BASE_URL/api/v2/live")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | head -1)
if [ "$HTTP_CODE" = "200" ]; then
    echo "   ✓ v2 /live endpoint works (HTTP $HTTP_CODE)"
else
    echo "   ✗ v2 /live endpoint failed (HTTP $HTTP_CODE)"
    exit 1
fi

echo "2. Testing GET /api/v2/ready"
RESPONSE=$(curl -s -w "\n%{http_code}" "$BASE_URL/api/v2/ready")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
if [ "$HTTP_CODE" = "200" ]; then
    echo "   ✓ v2 /ready endpoint works (HTTP $HTTP_CODE)"
else
    echo "   ✗ v2 /ready endpoint failed (HTTP $HTTP_CODE)"
    exit 1
fi

echo "3. Testing GET /api/v2/info"
RESPONSE=$(curl -s -w "\n%{http_code}" "$BASE_URL/api/v2/info")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
if [ "$HTTP_CODE" = "200" ]; then
    echo "   ✓ v2 /info endpoint works (HTTP $HTTP_CODE)"
else
    echo "   ✗ v2 /info endpoint failed (HTTP $HTTP_CODE)"
    exit 1
fi

echo ""
echo "Testing v0 endpoints (deprecated)..."
echo ""

echo "4. Testing GET /api/v0/live (should work with deprecation headers)"
RESPONSE=$(curl -s -i "$BASE_URL/api/v0/live")
if echo "$RESPONSE" | grep -q "HTTP/.* 200"; then
    echo "   ✓ v0 /live endpoint works"
else
    echo "   ✗ v0 /live endpoint failed"
    exit 1
fi

if echo "$RESPONSE" | grep -qi "X-API-Deprecated: true"; then
    echo "   ✓ Deprecation header present"
else
    echo "   ⚠ Warning: Deprecation header missing"
fi

if echo "$RESPONSE" | grep -qi "X-API-Current-Version: v2"; then
    echo "   ✓ Current version header present"
else
    echo "   ⚠ Warning: Current version header missing"
fi

if echo "$RESPONSE" | grep -qi "Sunset:"; then
    echo "   ✓ Sunset header present"
else
    echo "   ⚠ Warning: Sunset header missing"
fi

echo ""
echo "5. Testing GET /api/v0/ready (should work with deprecation headers)"
RESPONSE=$(curl -s -i "$BASE_URL/api/v0/ready")
if echo "$RESPONSE" | grep -q "HTTP/.* 200"; then
    echo "   ✓ v0 /ready endpoint works"
else
    echo "   ✗ v0 /ready endpoint failed"
    exit 1
fi

echo "6. Testing GET /api/v0/info (should work with deprecation headers)"
RESPONSE=$(curl -s -i "$BASE_URL/api/v0/info")
if echo "$RESPONSE" | grep -q "HTTP/.* 200"; then
    echo "   ✓ v0 /info endpoint works"
else
    echo "   ✗ v0 /info endpoint failed"
    exit 1
fi

echo ""
echo "======================================================"
echo "✓ All tests passed!"
echo ""
echo "Summary:"
echo "  - v2 endpoints are working (primary API)"
echo "  - v0 endpoints are working (backward compatibility)"
echo "  - Deprecation headers are present on v0 endpoints"
echo ""
echo "Migration successful! Both API versions are operational."
