# AWS Team Commands - FLAT (No Line Breaks)

## TABLE NAME: llm-proxy-api-keys--students-dev

Copy each command exactly as shown. NO line breaks.

---

## SETUP COMMANDS (Run in order, one at a time)

### Command 1: Create Table
```
aws dynamodb create-table --table-name llm-proxy-api-keys--students-dev --attribute-definitions AttributeName=pk,AttributeType=S --key-schema AttributeName=pk,KeyType=HASH --billing-mode PAY_PER_REQUEST --region us-west-2
```

### Command 2: Wait for Table
```
aws dynamodb wait table-exists --table-name llm-proxy-api-keys--students-dev --region us-west-2
```

### Command 3: List Tables (verify it exists)
```
aws dynamodb list-tables --region us-west-2
```

### Command 4: Grant IAM Permissions
```
aws iam put-user-policy --user-name bedrock-shethv --policy-name llm-proxy-dynamodb-access --policy-document '{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":["dynamodb:GetItem","dynamodb:PutItem","dynamodb:UpdateItem","dynamodb:DeleteItem","dynamodb:Query","dynamodb:Scan","dynamodb:BatchGetItem","dynamodb:BatchWriteItem"],"Resource":"arn:aws:dynamodb:us-west-2:*:table/llm-proxy-api-keys--students-dev"}]}' --region us-west-2
```

---

## VERIFICATION COMMANDS (Run these to confirm everything works)

### Verification 1: Describe Table
```
aws dynamodb describe-table --table-name llm-proxy-api-keys--students-dev --region us-west-2
```
Expected: "TableStatus": "ACTIVE"

### Verification 2: Check IAM Policy
```
aws iam get-user-policy --user-name bedrock-shethv --policy-name llm-proxy-dynamodb-access --region us-west-2
```
Expected: "Effect": "Allow"

### Verification 3: Test Write (Put Item)
```
aws dynamodb put-item --table-name llm-proxy-api-keys--students-dev --item '{"pk":{"S":"my:test-item"},"provider":{"S":"openai"},"actual_key":{"S":"sk-proj-test"},"daily_cost_limit":{"N":"50000"},"description":{"S":"Test Item"},"enabled":{"BOOL":true},"created_at":{"S":"2026-04-10T12:00:00Z"},"updated_at":{"S":"2026-04-10T12:00:00Z"}}' --region us-west-2
```
Expected: No error

### Verification 4: Test Read (Get Item)
```
aws dynamodb get-item --table-name llm-proxy-api-keys--students-dev --key '{"pk":{"S":"my:test-item"}}' --region us-west-2
```
Expected: Returns the item with all fields

### Verification 5: Test Scan (List Items)
```
aws dynamodb scan --table-name llm-proxy-api-keys--students-dev --region us-west-2
```
Expected: "Count": 1

### Verification 6: Test Update
```
aws dynamodb update-item --table-name llm-proxy-api-keys--students-dev --key '{"pk":{"S":"my:test-item"}}' --update-expression "SET description = :desc, updated_at = :now" --expression-attribute-values '{":desc":{"S":"Updated via verification"},":now":{"S":"2026-04-10T13:00:00Z"}}' --region us-west-2
```
Expected: No error

### Verification 7: Verify Update Worked
```
aws dynamodb get-item --table-name llm-proxy-api-keys--students-dev --key '{"pk":{"S":"my:test-item"}}' --region us-west-2
```
Expected: description shows "Updated via verification"

### Verification 8: Test Delete
```
aws dynamodb delete-item --table-name llm-proxy-api-keys--students-dev --key '{"pk":{"S":"my:test-item"}}' --region us-west-2
```
Expected: No error

### Verification 9: Verify Delete Worked
```
aws dynamodb get-item --table-name llm-proxy-api-keys--students-dev --key '{"pk":{"S":"my:test-item"}}' --region us-west-2
```
Expected: Empty response {}

### Verification 10: Test bedrock-shethv User Access (Run as bedrock-shethv on appmotel)
```
aws dynamodb scan --table-name llm-proxy-api-keys--students-dev --region us-west-2
```
Expected: No "AccessDenied" error

---

## COMPLETE SETUP + VERIFICATION SCRIPT (Copy-Paste All at Once)

```bash
aws dynamodb create-table --table-name llm-proxy-api-keys--students-dev --attribute-definitions AttributeName=pk,AttributeType=S --key-schema AttributeName=pk,KeyType=HASH --billing-mode PAY_PER_REQUEST --region us-west-2 && aws dynamodb wait table-exists --table-name llm-proxy-api-keys--students-dev --region us-west-2 && aws iam put-user-policy --user-name bedrock-shethv --policy-name llm-proxy-dynamodb-access --policy-document '{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":["dynamodb:GetItem","dynamodb:PutItem","dynamodb:UpdateItem","dynamodb:DeleteItem","dynamodb:Query","dynamodb:Scan","dynamodb:BatchGetItem","dynamodb:BatchWriteItem"],"Resource":"arn:aws:dynamodb:us-west-2:*:table/llm-proxy-api-keys--students-dev"}]}' --region us-west-2 && echo "Setup complete! Running verification..." && aws dynamodb describe-table --table-name llm-proxy-api-keys--students-dev --region us-west-2 | grep TableStatus && aws dynamodb put-item --table-name llm-proxy-api-keys--students-dev --item '{"pk":{"S":"my:verification"},"provider":{"S":"openai"},"actual_key":{"S":"sk-test"},"daily_cost_limit":{"N":"50000"},"description":{"S":"Verification"},"enabled":{"BOOL":true},"created_at":{"S":"2026-04-10T12:00:00Z"},"updated_at":{"S":"2026-04-10T12:00:00Z"}}' --region us-west-2 && aws dynamodb get-item --table-name llm-proxy-api-keys--students-dev --key '{"pk":{"S":"my:verification"}}' --region us-west-2 | grep -q "my:verification" && echo "✅ All tests passed! Table is ready."
```

---

## CLEANUP (If you need to start over)

```
aws dynamodb delete-table --table-name llm-proxy-api-keys--students-dev --region us-west-2 && aws dynamodb wait table-not-exists --table-name llm-proxy-api-keys--students-dev --region us-west-2 && aws iam delete-user-policy --user-name bedrock-shethv --policy-name llm-proxy-dynamodb-access --region us-west-2 && echo "Cleaned up!"
```

---

## CHECKLIST FOR AWS TEAM

After running all commands above, confirm each line works:

- [ ] Command 1: Create table - returns JSON with TableArn
- [ ] Command 2: Wait for table - completes without error
- [ ] Command 3: List tables - shows llm-proxy-api-keys--students-dev
- [ ] Command 4: Grant permissions - no error
- [ ] Verification 1: Describe table - shows "ACTIVE"
- [ ] Verification 2: Check IAM - shows policy with 8 actions
- [ ] Verification 3: Write - no error
- [ ] Verification 4: Read - returns item
- [ ] Verification 5: Scan - shows Count: 1
- [ ] Verification 6: Update - no error
- [ ] Verification 7: Verify update - description changed
- [ ] Verification 8: Delete - no error
- [ ] Verification 9: Verify delete - empty response
- [ ] Verification 10: bedrock-shethv access - no error

All checked? Report: ✅ READY FOR PRODUCTION

---

## EXACT TABLE NAME TO USE EVERYWHERE

**llm-proxy-api-keys--students-dev**

Note the double dash (--) between "keys" and "students"

---

## SUMMARY TO REPORT BACK

When everything is done, AWS team should provide this summary:

```
✅ DynamoDB Setup Complete

Table Name: llm-proxy-api-keys--students-dev
Table Status: ACTIVE
Table Region: us-west-2
IAM User: bedrock-shethv
IAM Policy: llm-proxy-dynamodb-access

Verification Results:
✅ Write: SUCCESS
✅ Read: SUCCESS
✅ Update: SUCCESS
✅ Delete: SUCCESS
✅ User Access: SUCCESS

Status: READY FOR PRODUCTION
```
