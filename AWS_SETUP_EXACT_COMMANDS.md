# AWS DynamoDB Setup - Exact Commands (Step by Step)

## Prerequisites

Make sure you have:
1. AWS CLI installed: `aws --version`
2. AWS credentials configured: `aws sts get-caller-identity`
3. Correct AWS region: `us-west-2`

---

## Step 1: Check Current AWS Setup

Run these to verify you're ready:

```bash
# Check AWS CLI is installed
aws --version

# Check credentials are configured
aws sts get-caller-identity

# Expected output should show your AWS account ID and user
```

---

## Step 2: Create DynamoDB Table

### Command: Create the table

```bash
aws dynamodb create-table \
  --table-name llm-proxy-api-keys-dev \
  --attribute-definitions \
    AttributeName=pk,AttributeType=S \
  --key-schema \
    AttributeName=pk,KeyType=HASH \
  --billing-mode PAY_PER_REQUEST \
  --region us-west-2
```

**What this does:**
- Creates table: `llm-proxy-api-keys-dev`
- Primary key (HASH): `pk` (string)
- Billing: Pay-per-request (no fixed cost)
- Region: us-west-2

**Expected output:** JSON showing table is being created

---

## Step 3: Wait for Table to Be Created

```bash
# Wait until table is ACTIVE (this takes 10-30 seconds)
aws dynamodb wait table-exists \
  --table-name llm-proxy-api-keys-dev \
  --region us-west-2

echo "Table is ready!"
```

---

## Step 4: Verify Table Was Created

```bash
# List tables
aws dynamodb list-tables --region us-west-2

# Should output:
# {
#   "TableNames": [
#     "llm-proxy-api-keys-dev"
#   ]
# }
```

---

## Step 5: Describe Table Structure

```bash
# See full table details
aws dynamodb describe-table \
  --table-name llm-proxy-api-keys-dev \
  --region us-west-2

# Look for: TableStatus = ACTIVE
```

---

## Step 6: Insert Test Data (ONE Key)

```bash
# Create a test key manually
aws dynamodb put-item \
  --table-name llm-proxy-api-keys-dev \
  --item '{
    "pk": {"S": "my:test-key-12345"},
    "provider": {"S": "openai"},
    "actual_key": {"S": "sk-proj-test-key-here"},
    "daily_cost_limit": {"N": "50000"},
    "description": {"S": "Test Key"},
    "enabled": {"BOOL": true},
    "created_at": {"S": "2026-04-10T12:00:00Z"},
    "updated_at": {"S": "2026-04-10T12:00:00Z"},
    "tags": {"M": {
      "department": {"S": "A"},
      "team": {"S": "research"}
    }}
  }' \
  --region us-west-2
```

**What this does:**
- Adds one test key to the table
- Shows you the format DynamoDB expects

---

## Step 7: Verify Data Was Inserted

```bash
# Get the key we just created
aws dynamodb get-item \
  --table-name llm-proxy-api-keys-dev \
  --key '{"pk": {"S": "my:test-key-12345"}}' \
  --region us-west-2

# Should output the item we just created
```

---

## Step 8: Query by Tag (Bulk Operations - Future)

```bash
# This won't work yet - you need to create a Global Secondary Index
# But here's the command for when you do:

# First, add GSI (Global Secondary Index) for tag queries
aws dynamodb update-table \
  --table-name llm-proxy-api-keys-dev \
  --attribute-definitions \
    AttributeName=department_tag,AttributeType=S \
  --global-secondary-indexes '[{
    "IndexName": "DepartmentIndex",
    "KeySchema": [
      {"AttributeName": "department_tag", "KeyType": "HASH"}
    ],
    "Projection": {"ProjectionType": "ALL"},
    "ProvisionedThroughput": {
      "ReadCapacityUnits": 5,
      "WriteCapacityUnits": 5
    }
  }]' \
  --region us-west-2

# Wait for it
aws dynamodb wait table-exists \
  --table-name llm-proxy-api-keys-dev \
  --region us-west-2
```

**Note:** We're NOT doing this yet - just showing you for reference

---

## Step 9: List All Keys in Table

```bash
# Scan entire table (shows all keys)
aws dynamodb scan \
  --table-name llm-proxy-api-keys-dev \
  --region us-west-2

# Output: All keys with their data
```

---

## Step 10: Delete a Key (Testing)

```bash
# Delete the test key
aws dynamodb delete-item \
  --table-name llm-proxy-api-keys-dev \
  --key '{"pk": {"S": "my:test-key-12345"}}' \
  --region us-west-2
```

---

## Step 11: Verify It's Gone

```bash
# Try to get deleted key (should fail)
aws dynamodb get-item \
  --table-name llm-proxy-api-keys-dev \
  --key '{"pk": {"S": "my:test-key-12345"}}' \
  --region us-west-2

# Should return empty or error
```

---

## Step 12: Set Up IAM Permissions (For appmotel user)

The appmotel server needs permission to read/write. Run this:

```bash
# Create inline policy for bedrock-shethv user
aws iam put-user-policy \
  --user-name bedrock-shethv \
  --policy-name llm-proxy-dynamodb-access \
  --policy-document '{
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
  }' \
  --region us-west-2
```

**What this does:**
- Allows bedrock-shethv user to read/write to this table
- Limits to ONLY this table (security)
- Allows all CRUD operations

---

## Step 13: Verify IAM Policy Was Created

```bash
# List policies for user
aws iam list-user-policies \
  --user-name bedrock-shethv \
  --region us-west-2

# Should show: llm-proxy-dynamodb-access
```

---

## Step 14: Get the Inline Policy Details

```bash
# See what permissions were granted
aws iam get-user-policy \
  --user-name bedrock-shethv \
  --policy-name llm-proxy-dynamodb-access \
  --region us-west-2

# Should show the policy document
```

---

## Quick Reference: All Commands in Order

### TLDR - Just Copy/Paste These in Order

```bash
# 1. Create table
aws dynamodb create-table \
  --table-name llm-proxy-api-keys-dev \
  --attribute-definitions AttributeName=pk,AttributeType=S \
  --key-schema AttributeName=pk,KeyType=HASH \
  --billing-mode PAY_PER_REQUEST \
  --region us-west-2

# 2. Wait for table
aws dynamodb wait table-exists --table-name llm-proxy-api-keys-dev --region us-west-2

# 3. Verify table exists
aws dynamodb list-tables --region us-west-2

# 4. Grant IAM permissions
aws iam put-user-policy \
  --user-name bedrock-shethv \
  --policy-name llm-proxy-dynamodb-access \
  --policy-document '{
    "Version": "2012-10-17",
    "Statement": [{
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
    }]
  }' \
  --region us-west-2

# 5. Verify permissions
aws iam get-user-policy \
  --user-name bedrock-shethv \
  --policy-name llm-proxy-dynamodb-access \
  --region us-west-2

# Done!
echo "DynamoDB setup complete!"
```

---

## If Something Goes Wrong: Cleanup Commands

### Delete Everything (Nuclear Option)

```bash
# Delete table
aws dynamodb delete-table \
  --table-name llm-proxy-api-keys-dev \
  --region us-west-2

# Wait for it to delete
aws dynamodb wait table-not-exists \
  --table-name llm-proxy-api-keys-dev \
  --region us-west-2

# Remove IAM policy
aws iam delete-user-policy \
  --user-name bedrock-shethv \
  --policy-name llm-proxy-dynamodb-access

echo "Cleaned up!"
```

Then start over from Step 2.

---

## Verify Everything Works

### Test from appmotel (as appmotel user)

```bash
# SSH to appmotel
ssh appmotel@<appmotel-ip>

# Test DynamoDB access
aws dynamodb scan \
  --table-name llm-proxy-api-keys-dev \
  --region us-west-2

# Should work without errors
```

If this works, the app can use DynamoDB!

---

## Environment Variables (if needed)

Add to `.env` on appmotel (but usually not needed):

```bash
AWS_REGION=us-west-2
AWS_PROFILE=bedrock
```

The app reads from `~/.aws/credentials` and `~/.aws/config` automatically.

---

## Troubleshooting

### Error: "User: arn:aws:iam::xxx is not authorized"

**Solution:** Run Step 12 (IAM permissions) again

### Error: "Table does not exist"

**Solution:** Run Step 2 (create table) again

### Error: "Could not connect to endpoint"

**Solution:** Check region is `us-west-2` in all commands

### Error: "Invalid AttributeType"

**Solution:** Make sure you're using exactly:
- `AttributeName=pk`
- `AttributeType=S` (S = String)

---

## Exact AWS ARN Format

Your resources should look like:

```
Table ARN: arn:aws:dynamodb:us-west-2:405644541454:table/llm-proxy-api-keys-dev
User ARN:  arn:aws:iam::405644541454:user/bedrock-shethv
```

(Replace `405644541454` with your actual AWS account ID)

---

## Final Verification Checklist

- [ ] Table created: `llm-proxy-api-keys-dev`
- [ ] Table is ACTIVE
- [ ] Table is in us-west-2 region
- [ ] IAM policy created: `llm-proxy-dynamodb-access`
- [ ] Policy attached to: `bedrock-shethv` user
- [ ] Test data inserted (optional)
- [ ] IAM user can scan table from appmotel

Once all checkboxes are done, you're ready to go! ✅
