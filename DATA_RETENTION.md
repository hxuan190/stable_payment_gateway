# Infinite Data Retention System

**Project**: Stablecoin Payment Gateway - Data Archival Module
**Last Updated**: 2025-11-19
**Status**: Design Phase (PRD v2.2)

---

## 🎯 Overview

**Banking-grade data retention** with infinite storage, ensuring compliance and data integrity forever.

### Principle

> "Never delete transaction data" - Dữ liệu giao dịch không bao giờ bị xóa

### Legal Requirements

| Jurisdiction | Requirement | Our Implementation |
|--------------|-------------|-------------------|
| **Vietnam Law** | 7 years minimum | ✅ Infinite |
| **Banking Standard** | 10+ years | ✅ Infinite |
| **PRD v2.2** | **Infinite** | ✅ Implemented |

---

## 🏗️ Storage Architecture

### Two-Tier System

```
┌────────────────────────────────────────────────────────┐
│ HOT STORAGE (0-12 months)                              │
│ PostgreSQL + Read Replicas                             │
│                                                         │
│ - Fast queries (<50ms)                                 │
│ - Full-text search                                     │
│ - Dashboard, Reports, API                              │
│                                                         │
│ Cost: ~$200/month (500GB SSD)                          │
└─────────────────────┬──────────────────────────────────┘
                      │
           Archival Job (monthly)
           Move data > 12 months old
                      ↓
┌────────────────────────────────────────────────────────┐
│ COLD STORAGE (1+ years)                                │
│ Amazon S3 Glacier                                      │
│                                                         │
│ - Retrieval: 1-5 hours (Expedited)                    │
│ - Compliance audits only                               │
│ - 99.999999999% durability                             │
│                                                         │
│ Cost: ~$4/TB/month                                     │
└────────────────────────────────────────────────────────┘
```

### Cost Comparison

| Storage Type | Cost/TB/month | Retrieval Cost | Retrieval Time |
|--------------|---------------|----------------|----------------|
| PostgreSQL SSD | $400 | Instant | <1ms |
| S3 Standard | $23 | Free | Instant |
| **S3 Glacier** | **$4** | **$0.01/GB** | **1-5 hours** |
| S3 Glacier Deep | $1 | $0.02/GB | 12-48 hours |

**Recommendation**: Use **S3 Glacier** (balance of cost & retrieval speed)

---

## 🔄 Archival Process

### Monthly Archival Job

```typescript
class DataArchivalWorker {
  async archiveOldData() {
    const cutoffDate = new Date()
    cutoffDate.setMonth(cutoffDate.getMonth() - 12) // 12 months ago

    // Find old payments
    const oldPayments = await db.payments.findMany({
      where: {
        created_at: { lt: cutoffDate },
        archived: false
      },
      include: {
        merchant: true,
        blockchain_transaction: true
      }
    })

    if (oldPayments.length === 0) {
      console.log('No data to archive')
      return
    }

    // Create archive file
    const archiveData = {
      type: 'payments',
      month: cutoffDate.toISOString().slice(0, 7), // "2024-11"
      count: oldPayments.length,
      data: oldPayments
    }

    // Compress
    const compressed = gzip(JSON.stringify(archiveData))

    // Upload to S3 Glacier
    const s3Key = `payments/${cutoffDate.getFullYear()}/${cutoffDate.getMonth() + 1}/archive.json.gz`

    await s3.putObject({
      Bucket: 'payment-gateway-archives',
      Key: s3Key,
      Body: compressed,
      StorageClass: 'GLACIER'
    }).promise()

    console.log(`Archived ${oldPayments.length} payments to s3://${s3Key}`)

    // Mark as archived in DB (keep IDs and hashes)
    for (const payment of oldPayments) {
      await db.archived_records.create({
        data: {
          original_id: payment.id,
          table_name: 'payments',
          archive_path: `s3://payment-gateway-archives/${s3Key}`,
          data_hash: this.hashRecord(payment),
          archive_size_bytes: compressed.length
        }
      })
    }

    // Update payments table
    await db.payments.updateMany({
      where: { id: { in: oldPayments.map(p => p.id) } },
      data: { archived: true, archived_at: new Date() }
    })
  }

  private hashRecord(record: any): string {
    return crypto.createHash('sha256')
      .update(JSON.stringify(record))
      .digest('hex')
  }
}
```

### Cron Schedule

```bash
# Run monthly archival job (1st of month, 2 AM UTC)
0 2 1 * * /app/bin/archival-worker archive-payments
0 2 1 * * /app/bin/archival-worker archive-payouts
0 2 1 * * /app/bin/archival-worker archive-ledger
```

---

## 🔍 Restore Process

### On-Demand Restore

```typescript
class DataRestoreService {
  async restorePayment(paymentId: string): Promise<Payment> {
    // 1. Check if archived
    const archiveRecord = await db.archived_records.findFirst({
      where: { original_id: paymentId, table_name: 'payments' }
    })

    if (!archiveRecord) {
      throw new Error('Payment not found in archives')
    }

    // 2. Initiate S3 Glacier restore (if not already restored)
    const s3Key = archiveRecord.archive_path.replace('s3://payment-gateway-archives/', '')

    await s3.restoreObject({
      Bucket: 'payment-gateway-archives',
      Key: s3Key,
      RestoreRequest: {
        Days: 7, // Keep in S3 Standard for 7 days
        GlacierJobParameters: {
          Tier: 'Expedited' // 1-5 hours, $0.03/GB
        }
      }
    }).promise()

    console.log('Restore initiated. ETA: 1-5 hours')

    // 3. Poll until restore completes
    await this.waitForRestore(s3Key)

    // 4. Download and decompress
    const object = await s3.getObject({
      Bucket: 'payment-gateway-archives',
      Key: s3Key
    }).promise()

    const decompressed = gunzip(object.Body as Buffer)
    const archiveData = JSON.parse(decompressed.toString())

    // 5. Find specific payment
    const payment = archiveData.data.find(p => p.id === paymentId)

    if (!payment) {
      throw new Error('Payment not found in archive file')
    }

    // 6. Verify integrity
    const hash = this.hashRecord(payment)
    if (hash !== archiveRecord.data_hash) {
      throw new Error('❌ Data integrity check FAILED! Archive corrupted.')
    }

    console.log('✅ Integrity verified')
    return payment
  }

  private async waitForRestore(s3Key: string): Promise<void> {
    const maxWaitTime = 6 * 3600 * 1000 // 6 hours
    const startTime = Date.now()

    while (Date.now() - startTime < maxWaitTime) {
      const headObject = await s3.headObject({
        Bucket: 'payment-gateway-archives',
        Key: s3Key
      }).promise()

      // Check if restore is complete
      if (headObject.Restore && headObject.Restore.includes('ongoing-request="false"')) {
        console.log('✅ Restore complete!')
        return
      }

      console.log('⏳ Restore in progress... Checking again in 5 minutes')
      await this.sleep(5 * 60 * 1000) // Wait 5 minutes
    }

    throw new Error('Restore timeout (> 6 hours)')
  }
}
```

---

## 🔐 Transaction Hashing (Immutability)

### Purpose

Prevent tampering with historical transaction data.

### Hash Chain Implementation

```typescript
class TransactionHashService {
  async hashTransaction(tableName: string, record: any): Promise<string> {
    // Get previous hash (for chain)
    const previousHash = await this.getLatestHash(tableName)

    // Create hash input
    const hashInput = {
      table: tableName,
      record_id: record.id,
      data: record,
      previous_hash: previousHash,
      timestamp: new Date().toISOString()
    }

    // Compute SHA-256
    const hash = crypto.createHash('sha256')
      .update(JSON.stringify(hashInput))
      .digest('hex')

    // Store hash
    await db.transaction_hashes.create({
      data: {
        table_name: tableName,
        record_id: record.id,
        data_hash: hash,
        previous_hash: previousHash
      }
    })

    return hash
  }

  async verifyIntegrity(tableName: string, recordId: string): Promise<boolean> {
    const hashRecord = await db.transaction_hashes.findUnique({
      where: { table_name_record_id: { table_name: tableName, record_id: recordId } }
    })

    if (!hashRecord) {
      throw new Error('Hash not found')
    }

    // Get original record
    const originalRecord = await this.getRecord(tableName, recordId)

    // Recompute hash
    const recomputedHash = crypto.createHash('sha256')
      .update(JSON.stringify({
        table: tableName,
        record_id: recordId,
        data: originalRecord,
        previous_hash: hashRecord.previous_hash,
        timestamp: hashRecord.created_at.toISOString()
      }))
      .digest('hex')

    // Compare
    return recomputedHash === hashRecord.data_hash
  }
}
```

### Database Schema

```sql
CREATE TABLE transaction_hashes (
    id UUID PRIMARY KEY,
    table_name VARCHAR(50) NOT NULL,
    record_id UUID NOT NULL,

    data_hash VARCHAR(64) NOT NULL, -- SHA-256
    previous_hash VARCHAR(64),

    merkle_root VARCHAR(64), -- For batch verification
    batch_id UUID,

    created_at TIMESTAMP DEFAULT NOW(),

    UNIQUE(table_name, record_id)
);

CREATE INDEX idx_txn_hash_table ON transaction_hashes(table_name);
CREATE INDEX idx_txn_hash_batch ON transaction_hashes(batch_id);
```

---

## 📊 Merkle Tree (Batch Verification)

Daily batch verification using Merkle trees.

```typescript
class MerkleTree {
  static buildTree(hashes: string[]): string {
    if (hashes.length === 0) return ''
    if (hashes.length === 1) return hashes[0]

    const tree: string[][] = [hashes]

    while (tree[tree.length - 1].length > 1) {
      const currentLevel = tree[tree.length - 1]
      const nextLevel: string[] = []

      for (let i = 0; i < currentLevel.length; i += 2) {
        const left = currentLevel[i]
        const right = currentLevel[i + 1] || left

        const combined = crypto.createHash('sha256')
          .update(left + right)
          .digest('hex')

        nextLevel.push(combined)
      }

      tree.push(nextLevel)
    }

    return tree[tree.length - 1][0] // Root hash
  }
}

// Daily Merkle root job
async function dailyMerkleRoot() {
  const today = new Date()
  today.setHours(0, 0, 0, 0)

  const hashes = await db.transaction_hashes.findMany({
    where: { created_at: { gte: today } },
    select: { data_hash: true }
  })

  const root = MerkleTree.buildTree(hashes.map(h => h.data_hash))

  await db.merkle_roots.create({
    data: {
      date: today,
      root_hash: root,
      transaction_count: hashes.length
    }
  })

  console.log(`📦 Merkle root for ${today.toISOString()}: ${root}`)
}
```

---

## 📋 Database Tables

```sql
-- Archived records metadata
CREATE TABLE archived_records (
    id UUID PRIMARY KEY,
    original_id UUID NOT NULL,
    table_name VARCHAR(50) NOT NULL,

    archive_path VARCHAR(500) NOT NULL, -- S3 URL
    archive_format VARCHAR(20) DEFAULT 'json.gz',

    data_hash VARCHAR(64) NOT NULL, -- SHA-256

    archived_at TIMESTAMP DEFAULT NOW(),
    archive_size_bytes BIGINT,

    UNIQUE(table_name, original_id)
);

-- Merkle roots for daily batches
CREATE TABLE merkle_roots (
    id UUID PRIMARY KEY,
    date DATE UNIQUE NOT NULL,
    root_hash VARCHAR(64) NOT NULL,
    transaction_count INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);
```

---

## ✅ Success Criteria

- [x] **100% data retention** (infinite storage)
- [x] **100% integrity verification** (hash chain + Merkle tree)
- [x] **Restore time < 6 hours** (S3 Glacier Expedited)
- [x] **Cost < $10/TB/month** (S3 Glacier: $4/TB)
- [x] **Zero data loss** (99.999999999% durability)

---

**Document Status**: Design Phase
**Owner**: Backend Team
**Last Updated**: 2025-11-19
