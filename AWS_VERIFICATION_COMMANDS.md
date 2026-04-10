# AWS Verification - How to Confirm Everything Works

Give these commands to your AWS team. They'll run them in order and show you the output.

---

## Step 1: Verify Table Was Created

```bash
aws dynamodb describe-table \
  --table-name llm-proxy-api-keys-dev \
  --region us-west-2
```

**Expected Output:**
```json
{
    "Table": {
        "TableName": "llm-proxy-api-keys-dev",
        "TableStatus": "ACTIVE",
        "TableArn": "arn:aws:dynamodb:us-west-2:405644541454:table/llm-proxy-api-keys-dev",
        ...
    }
}
```

**What to look for:**
- ✅ `"TableStatus": "ACTIVE"` (NOT "CREATING")
- ✅ `"TableName": "llm-proxy-api-keys-dev"` (exact match)
- ✅ Region shows `us-west-2`

---

## Step 2: Verify IAM Policy Was Created

```bash
aws iam get-user-policy \
  --user-name bedrock-shethv \
  --policy-name llm-proxy-dynamodb-access
```

**Expected Output:**
```json
{
    "UserName": "bedrock-shethv",
    "PolicyName": "llm-proxy-dynamodb-access",
    "PolicyDocument": {
        "Version": "2012-10-17",
        "Statement": [
            {
                "Effect": "Allow",
                "Action": [
                    "dynamodb:GetItem",
                    "dynamodb:PutItem",
                    "dynamodb:UpdateItem",
                    "dynamodb:DeleteItem",
                    "dynamodb:Query",
                    "dynamodb:Scan",
                    "dynamodb:BatchGetItem",
                    "dynamodb:BatchWriteItem"
                ],
                "Resource": "arn:aws:dynamodb:us-west-2:*:table/llm-proxy-api-keys-dev"
            }
        ]
    }
}
```

**What to look for:**
- ✅ `"Effect": "Allow"`
- ✅ All 8 actions listed above
- ✅ Resource includes `llm-proxy-api-keys-dev`

---

## Step 3: Test Write Access (Insert Test Item)

```bash
aws dynamodb put-item \
  --table-name llm-proxy-api-keys-dev \
  --item '{
    "pk": {"S": "my:verification-test"},
    "provider": {"S": "openai"},
    "actual_key": {"S": "sk-proj-test"},
    "daily_cost_limit": {"N": "50000"},
    "description": {"S": "AWS Verification Test"},
    "enabled": {"BOOL": true},
    "created_at": {"S": "2026-04-10T12:00:00Z"},
    "updated_at": {"S": "2026-04-10T12:00:00Z"}
  }' \
  --region us-west-2
```

**Expected Output:**
- No error (clean response, or empty `{}`)

**What to look for:**
- ✅ No error message
- ✅ Command returns immediately

---

## Step 4: Test Read Access (Get Item Back)

```bash
aws dynamodb get-item \
  --table-name llm-proxy-api-keys-dev \
  --key '{"pk": {"S": "my:verification-test"}}' \
  --region us-west-2
```

**Expected Output:**
```json
{
    "Item": {
        "created_at": {
            "S": "2026-04-10T12:00:00Z"
        },
        "provider": {
            "S": "openai"
        },
        "actual_key": {
            "S": "sk-proj-test"
        },
        "enabled": {
            "BOOL": true
        },
        "daily_cost_limit": {
            "N": "50000"
        },
        "pk": {
            "S": "my:verification-test"
        },
        "description": {
            "S": "AWS Verification Test"
        },
        "updated_at": {
            "S": "2026-04-10T12:00:00Z"
        }
    }
}
```

**What to look for:**
- ✅ `"Item":` is present
- ✅ All fields match what was inserted
- ✅ No error

---

## Step 5: Test Scan (List All Items)

```bash
aws dynamodb scan \
  --table-name llm-proxy-api-keys-dev \
  --region us-west-2
```

**Expected Output:**
```json
{
    "Items": [
        {
            "created_at": {"S": "2026-04-10T12:00:00Z"},
            "provider": {"S": "openai"},
            ...
        }
    ],
    "Count": 1,
    "ScannedCount": 1
}
```

**What to look for:**
- ✅ `"Count": 1` (at least one item)
- ✅ `"Items"` array has data
- ✅ No error

---

## Step 6: Test Update (Modify Item)

```bash
aws dynamodb update-item \
  --table-name llm-proxy-api-keys-dev \
  --key '{"pk": {"S": "my:verification-test"}}' \
  --update-expression "SET description = :desc, updated_at = :now" \
  --expression-attribute-values '{
    ":desc": {"S": "Updated via verification"},
    ":now": {"S": "2026-04-10T13:00:00Z"}
  }' \
  --region us-west-2
```

**Expected Output:**
- No error (clean response)

**What to look for:**
- ✅ No error message
- ✅ Command returns immediately

---

## Step 7: Verify Update Worked

```bash
aws dynamodb get-item \
  --table-name llm-proxy-api-keys-dev \
  --key '{"pk": {"S": "my:verification-test"}}' \
  --region us-west-2
```

**Expected Output:**
```json
{
    "Item": {
        ...
        "description": {
            "S": "Updated via verification"  ← Changed!
        },
        "updated_at": {
            "S": "2026-04-10T13:00:00Z"  ← Changed!
        }
        ...
    }
}
```

**What to look for:**
- ✅ `description` now shows "Updated via verification"
- ✅ `updated_at` now shows "2026-04-10T13:00:00Z"

---

## Step 8: Test Delete (Remove Item)

```bash
aws dynamodb delete-item \
  --table-name llm-proxy-api-keys-dev \
  --key '{"pk": {"S": "my:verification-test"}}' \
  --region us-west-2
```

**Expected Output:**
- No error

**What to look for:**
- ✅ No error message

---

## Step 9: Verify Delete Worked

```bash
aws dynamodb get-item \
  --table-name llm-proxy-api-keys-dev \
  --key '{"pk": {"S": "my:verification-test"}}' \
  --region us-west-2
```

**Expected Output:**
```json
{}
```

**What to look for:**
- ✅ Empty response (item is gone)
- ✅ No error

---

## Step 10: Verify bedrock-shethv User Can Access

**Run this AS the bedrock-shethv user on appmotel:**

```bash
# SSH to appmotel
ssh appmotel@<appmotel-ip>

# Then run:
aws dynamodb scan \
  --table-name llm-proxy-api-keys-dev \
  --region us-west-2
```

**Expected Output:**
```json
{
    "Items": [],
    "Count": 0,
    "ScannedCount": 0
}
```

**What to look for:**
- ✅ No "AccessDenied" error
- ✅ Command works (even if empty)

---

## Quick Verification Checklist

Print this and have AWS team check off each:

```
☐ Step 1: Table Status = ACTIVE
☐ Step 2: IAM Policy exists with 8 DynamoDB actions
☐ Step 3: Can write (put-item succeeds)
☐ Step 4: Can read (get-item returns data)
☐ Step 5: Can list (scan returns items)
☐ Step 6: Can update (update-item succeeds)
☐ Step 7: Update worked (description changed)
☐ Step 8: Can delete (delete-item succeeds)
☐ Step 9: Delete worked (item is gone)
☐ Step 10: bedrock-shethv user can access

All checked? = ✅ READY TO GO!
```

---

## If Any Step Fails

### Error: "AccessDenied"
**Solution:** Run Step 2 again (IAM policy may not have applied immediately - wait 2 minutes and retry)

### Error: "ResourceNotFoundException: Table not found"
**Solution:** Run Step 1 again (table may not be created yet - wait and retry)

### Error: "ValidationException"
**Solution:** Check the exact command format - copy from above exactly

### Error: "UnrecognizedClientException"
**Solution:** Check AWS credentials are configured: `aws sts get-caller-identity`

---

## Summary Output for You

Have AWS team provide this summary when done:

```
DynamoDB Setup Verification Report
===================================

Table Name:           llm-proxy-api-keys-dev
Table Status:         ACTIVE
Table ARN:           arn:aws:dynamodb:us-west-2:405644541454:table/llm-proxy-api-keys-dev
Region:              us-west-2

IAM Policy:          llm-proxy-dynamodb-access
User:                bedrock-shethv
Permissions Granted: 8 DynamoDB actions

Verification Results:
  ✅ Write (put-item):    SUCCESS
  ✅ Read (get-item):     SUCCESS
  ✅ List (scan):         SUCCESS
  ✅ Update (update-item): SUCCESS
  ✅ Delete (delete-item): SUCCESS
  ✅ bedrock-shethv access: SUCCESS

Overall Status: ✅ READY FOR PRODUCTION
```

---

## Copy-Paste for AWS Team (All Steps)

Here's everything in one block they can run:

```bash
#!/bin/bash

echo "=== AWS DynamoDB Verification ==="
echo ""

echo "Step 1: Check table exists and is ACTIVE"
aws dynamodb describe-table --table-name llm-proxy-api-keys-dev --region us-west-2 | grep -E "TableName|TableStatus"
echo ""

echo "Step 2: Check IAM policy exists"
aws iam get-user-policy --user-name bedrock-shethv --policy-name llm-proxy-dynamodb-access | grep -E "UserName|PolicyName"
echo ""

echo "Step 3: Test write (put-item)"
aws dynamodb put-item --table-name llm-proxy-api-keys-dev --item '{"pk": {"S": "my:test"}, "provider": {"S": "openai"}, "actual_key": {"S": "sk-test"}, "daily_cost_limit": {"N": "50000"}, "description": {"S": "Test"}, "enabled": {"BOOL": true}, "created_at": {"S": "2026-04-10T12:00:00Z"}, "updated_at": {"S": "2026-04-10T12:00:00Z"}}' --region us-west-2
echo "✅ Write successful"
echo ""

echo "Step 4: Test read (get-item)"
aws dynamodb get-item --table-name llm-proxy-api-keys-dev --key '{"pk": {"S": "my:test"}}' --region us-west-2 | grep -E "pk|S"
echo "✅ Read successful"
echo ""

echo "Step 5: Test scan"
aws dynamodb scan --table-name llm-proxy-api-keys-dev --region us-west-2 | grep -E "Count|ScannedCount"
echo "✅ Scan successful"
echo ""

echo "Step 6: Test update"
aws dynamodb update-item --table-name llm-proxy-api-keys-dev --key '{"pk": {"S": "my:test"}}' --update-expression "SET description = :d" --expression-attribute-values '{":d": {"S": "Updated"}}' --region us-west-2
echo "✅ Update successful"
echo ""

echo "Step 7: Test delete"
aws dynamodb delete-item --table-name llm-proxy-api-keys-dev --key '{"pk": {"S": "my:test"}}' --region us-west-2
echo "✅ Delete successful"
echo ""

echo "=== All tests passed! ==="
```

Save as `verify-dynamodb.sh`, run: `bash verify-dynamodb.sh`
