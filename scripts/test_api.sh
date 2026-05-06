#!/bin/bash
# QR Restaurant API Test Script
# Usage: bash scripts/test_api.sh

set -euo pipefail

BASE="${API_BASE:-http://localhost:8080/api/v1}"
TOKEN=""
RID=""
TABLE_ID=""
QR_TOKEN=""
SESSION_TOKEN=""
ITEM_ID=""
ORDER_ID=""

echo "=== QR Restaurant API Test Suite ==="
echo "Base URL: $BASE"
echo ""

# --- Auth ---
echo "--- 1. Register Owner ---"
REG_RESP=$(curl -s -X POST "$BASE/auth/register" \
  -H "Content-Type: application/json" \
  -d '{"email":"owner@test.com","password":"TestPass123!","role":"OWNER"}')
echo "$REG_RESP" | python3 -m json.tool 2>/dev/null || echo "$REG_RESP"
echo ""

echo "--- 2. Register (duplicate - expect 409) ---"
curl -s -X POST "$BASE/auth/register" \
  -H "Content-Type: application/json" \
  -d '{"email":"owner@test.com","password":"TestPass123!","role":"OWNER"}'
echo ""
sleep 1

echo "--- 3. Login ---"
LOGIN_RESP=$(curl -s -X POST "$BASE/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"email":"owner@test.com","password":"TestPass123!"}')
TOKEN=$(echo "$LOGIN_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['access_token'])" 2>/dev/null)
echo "Token obtained: ${TOKEN:0:20}..."
echo ""

echo "--- 4. Get Me ---"
curl -s "$BASE/auth/me" -H "Authorization: Bearer $TOKEN" | python3 -m json.tool 2>/dev/null
echo ""

echo "--- 5. Create Restaurant ---"
REST_RESP=$(curl -s -X POST "$BASE/restaurants" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Pizza Palace","slug":"pizza-palace","description":"Best pizza in town","address":"123 Main St","phone":"+94771234567"}')
RID=$(echo "$REST_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['id'])" 2>/dev/null)
echo "Restaurant ID: $RID"
echo "$REST_RESP" | python3 -m json.tool 2>/dev/null
echo ""

echo "--- 6. List Owned Restaurants ---"
curl -s "$BASE/restaurants" -H "Authorization: Bearer $TOKEN" | python3 -m json.tool 2>/dev/null
echo ""

echo "--- 7. Get Restaurant ---"
curl -s "$BASE/restaurants/$RID" -H "Authorization: Bearer $TOKEN" | python3 -m json.tool 2>/dev/null
echo ""

# --- Staff ---
echo "--- 8. Create Staff ---"
curl -s -X POST "$BASE/restaurants/$RID/staff" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"email":"staff@pizza.com","password":"StaffPass123!"}' | python3 -m json.tool 2>/dev/null
echo ""

echo "--- 9. List Staff ---"
curl -s "$BASE/restaurants/$RID/staff" -H "Authorization: Bearer $TOKEN" | python3 -m json.tool 2>/dev/null
echo ""

# --- Theme ---
echo "--- 10. Get Theme ---"
curl -s "$BASE/restaurants/$RID/theme" -H "Authorization: Bearer $TOKEN" | python3 -m json.tool 2>/dev/null
echo ""

echo "--- 11. Update Theme ---"
curl -s -X PUT "$BASE/restaurants/$RID/theme" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"primary_color":"#E63946","secondary_color":"#F1FAEE","font_style":"poppins","dark_mode":true}' | python3 -m json.tool 2>/dev/null
echo ""

# --- Tables ---
echo "--- 12. Create Table T01 ---"
TABLE_RESP=$(curl -s -X POST "$BASE/restaurants/$RID/tables" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"table_code":"T01"}')
TABLE_ID=$(echo "$TABLE_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['id'])" 2>/dev/null)
QR_TOKEN=$(echo "$TABLE_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['qr_token'])" 2>/dev/null)
echo "Table ID: $TABLE_ID, QR Token: $QR_TOKEN"
echo "$TABLE_RESP" | python3 -m json.tool 2>/dev/null
echo ""

echo "--- 13. Create Table T02 ---"
curl -s -X POST "$BASE/restaurants/$RID/tables" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"table_code":"T02"}' | python3 -m json.tool 2>/dev/null
echo ""

echo "--- 14. List Tables ---"
curl -s "$BASE/restaurants/$RID/tables" -H "Authorization: Bearer $TOKEN" | python3 -m json.tool 2>/dev/null
echo ""

# --- Menu ---
echo "--- 15. Create Category: Starters ---"
CAT1=$(curl -s -X POST "$BASE/restaurants/$RID/menu/categories" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Starters","sort_order":1}')
CAT1_ID=$(echo "$CAT1" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['id'])" 2>/dev/null)
echo "Category ID: $CAT1_ID"
echo ""

echo "--- 16. Create Category: Mains ---"
CAT2=$(curl -s -X POST "$BASE/restaurants/$RID/menu/categories" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Mains","sort_order":2}')
CAT2_ID=$(echo "$CAT2" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['id'])" 2>/dev/null)
echo "Category ID: $CAT2_ID"
echo ""

echo "--- 17. Create Item: Garlic Bread (in Starters) ---"
ITEM1=$(curl -s -X POST "$BASE/restaurants/$RID/menu/categories/$CAT1_ID/items" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Garlic Bread","description":"Crispy garlic bread","price_cents":400,"sort_order":1}')
ITEM1_ID=$(echo "$ITEM1" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['id'])" 2>/dev/null)
echo "Item ID: $ITEM1_ID"
echo ""

echo "--- 18. Create Item: Margherita Pizza (in Mains) ---"
ITEM2=$(curl -s -X POST "$BASE/restaurants/$RID/menu/categories/$CAT2_ID/items" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Margherita Pizza","description":"Classic tomato and mozzarella","price_cents":1200,"sort_order":1}')
ITEM2_ID=$(echo "$ITEM2" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['id'])" 2>/dev/null)
echo "Item ID: $ITEM2_ID"
echo ""

echo "--- 19. List Categories ---"
curl -s "$BASE/restaurants/$RID/menu/categories" -H "Authorization: Bearer $TOKEN" | python3 -m json.tool 2>/dev/null
echo ""

echo "--- 20. List Items ---"
curl -s "$BASE/restaurants/$RID/menu/items" -H "Authorization: Bearer $TOKEN" | python3 -m json.tool 2>/dev/null
echo ""

# --- Public Customer Flow ---
echo ""
echo "=== Customer Flow ==="

echo "--- 21. Scan QR ---"
SCAN_RESP=$(curl -s "$BASE/public/scan?token=$QR_TOKEN")
SESSION_TOKEN=$(echo "$SCAN_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['session_token'])" 2>/dev/null)
echo "Session Token: $SESSION_TOKEN"
echo "$SCAN_RESP" | python3 -m json.tool 2>/dev/null
echo ""

echo "--- 22. Get Public Menu ---"
curl -s "$BASE/public/sessions/$SESSION_TOKEN/menu" | python3 -m json.tool 2>/dev/null
echo ""

echo "--- 23. Place Order ---"
ORDER_RESP=$(curl -s -X POST "$BASE/public/sessions/$SESSION_TOKEN/orders" \
  -H "Content-Type: application/json" \
  -d "{\"items\":[{\"menu_item_id\":\"$ITEM1_ID\",\"quantity\":2},{\"menu_item_id\":\"$ITEM2_ID\",\"quantity\":1}],\"notes\":\"Extra cheese please\"}")
ORDER_ID=$(echo "$ORDER_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['id'])" 2>/dev/null)
echo "Order ID: $ORDER_ID"
echo "$ORDER_RESP" | python3 -m json.tool 2>/dev/null
echo ""

echo "--- 24. Poll Customer Orders ---"
curl -s "$BASE/public/sessions/$SESSION_TOKEN/orders" | python3 -m json.tool 2>/dev/null
echo ""

echo "--- 25. Poll Session Status ---"
curl -s "$BASE/public/sessions/$SESSION_TOKEN" | python3 -m json.tool 2>/dev/null
echo ""

# --- Admin Order Management ---
echo ""
echo "=== Admin Order Management ==="

echo "--- 26. Admin List Orders ---"
curl -s "$BASE/restaurants/$RID/orders" -H "Authorization: Bearer $TOKEN" | python3 -m json.tool 2>/dev/null
echo ""

echo "--- 27. Update Order Status: PENDING -> ACCEPTED ---"
curl -s -X PATCH "$BASE/restaurants/$RID/orders/$ORDER_ID/status" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"status":"ACCEPTED"}' | python3 -m json.tool 2>/dev/null
echo ""

echo "--- 28. Update Order Status: ACCEPTED -> PREPARING ---"
curl -s -X PATCH "$BASE/restaurants/$RID/orders/$ORDER_ID/status" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"status":"PREPARING"}' | python3 -m json.tool 2>/dev/null
echo ""

echo "--- 29. Admin List Sessions ---"
curl -s "$BASE/restaurants/$RID/sessions" -H "Authorization: Bearer $TOKEN" | python3 -m json.tool 2>/dev/null
echo ""

echo "--- 30. Get Session Detail ---"
SESSION_ID=$(curl -s "$BASE/restaurants/$RID/sessions" -H "Authorization: Bearer $TOKEN" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d['data'][0]['id'])" 2>/dev/null)
curl -s "$BASE/restaurants/$RID/sessions/$SESSION_ID" -H "Authorization: Bearer $TOKEN" | python3 -m json.tool 2>/dev/null
echo ""

echo "--- 31. Close Session (should fail - orders exist) ---"
curl -s -X POST "$BASE/restaurants/$RID/sessions/$SESSION_ID/close" \
  -H "Authorization: Bearer $TOKEN" | python3 -m json.tool 2>/dev/null
echo ""

echo "--- 32. Force Close Session ---"
curl -s -X POST "$BASE/restaurants/$RID/sessions/$SESSION_ID/close" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"force":true}' | python3 -m json.tool 2>/dev/null
echo ""

# --- Auth Tests ---
echo ""
echo "=== Auth Tests ==="

echo "--- 33. Access without token (expect 401) ---"
curl -s "$BASE/restaurants" | python3 -m json.tool 2>/dev/null
echo ""

echo "--- 34. Staff Login ---"
STAFF_LOGIN=$(curl -s -X POST "$BASE/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"email":"staff@pizza.com","password":"StaffPass123!"}')
STAFF_TOKEN=$(echo "$STAFF_LOGIN" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['access_token'])" 2>/dev/null)
echo ""

echo "--- 35. Staff view tables (should succeed) ---"
curl -s "$BASE/restaurants/$RID/tables" -H "Authorization: Bearer $STAFF_TOKEN" | python3 -m json.tool 2>/dev/null
echo ""

echo "--- 36. Staff create table (should fail - OWNER only) ---"
curl -s -X POST "$BASE/restaurants/$RID/tables" \
  -H "Authorization: Bearer $STAFF_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"table_code":"T03"}' | python3 -m json.tool 2>/dev/null
echo ""

# --- Health Check ---
echo ""
echo "--- 37. Health Check ---"
curl -s "$BASE/../health" 2>/dev/null || curl -s "http://localhost:8080/api/v1/health"
echo ""

echo ""
echo "=== Test Suite Complete ==="
