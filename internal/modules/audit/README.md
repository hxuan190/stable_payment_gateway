# Audit Service

## 1. Overview
The Audit Service is a centralized module responsible for recording, storing, and retrieving immutable audit logs for all critical system actions. It serves as the "source of truth" for compliance, security investigations, and operational debugging by tracking **who** did **what**, **when**, and **with what result**.

**Responsibility**:
-   **Immutable Record Keeping**: Persists a tamper-evident trail of all state-changing operations.
-   **Compliance & Security**: Provides the data foundation for AML/KYC audits and security forensics.
-   **State Change Tracking**: Captures "before" and "after" snapshots of modified resources.

## 2. Architecture & Flow

The Audit Service operates as a write-heavy, append-only system.

```mermaid
graph TD
    subgraph Clients
        API[API Gateway]
        Worker[Background Worker]
        Admin[Admin Panel]
    end

    subgraph Audit_Module
        Entry[Entry Point]
        Repo[Audit Repository]
    end

    subgraph Storage
        DB[(PostgreSQL)]
    end

    API -->|Action Event| Entry
    Worker -->|System Event| Entry
    Admin -->|Admin Action| Entry

    Entry -->|Validate & Enrich| Repo
    Repo -->|INSERT (Append Only)| DB

    note[Immutable Log Table]
    DB -.- note
```

### Data Flow
1.  **Event Generation**: Any service (Payment, Merchant, etc.) generates an audit event upon completing (or failing) a significant action.
2.  **Enrichment**: The event is populated with actor details (ID, IP, Email), context (Request ID), and state changes (Old/New Values).
3.  **Persistence**: The `AuditRepository` inserts the record into the `audit_logs` table. This operation is strictly **INSERT-only**.
4.  **Retrieval**: Auditors or admins query the logs via strict filters (Time range, Actor, Resource).

## 3. Key Components

### Interfaces & Structs
-   **`Module`**: The main entry point that initializes the repository and logger.
-   **`AuditLog` (Domain)**: The core struct representing a single audit entry. It contains:
    -   **Actor**: `ActorType`, `ActorID`, `ActorIPAddress`.
    -   **Action**: `Action`, `ActionCategory` (e.g., `payment`, `kyc`), `Status`.
    -   **Resource**: `ResourceType`, `ResourceID`.
    -   **Context**: `RequestID`, `CorrelationID`.
    -   **State**: `OldValues`, `NewValues` (JSONB).
-   **`AuditRepository`**: Handles direct database interactions.
    -   `Create(log)`: Persists a single log.
    -   `CreateBatch(logs)`: Optimized batch insertion for high-throughput scenarios.
    -   `List(filter)`: Complex querying with `AuditFilter`.

### Critical Functions
-   **`Create`**: Ensures that the `CreatedAt` timestamp is set to the current server time (preventing timestamp spoofing) and persists the log.
-   **`List`**: Implements a flexible filtering mechanism allowing queries by almost any field (Actor, Action, Time Range, Status), essential for forensic investigations.
-   **`GetRecentFailures`**: A specialized query to quickly identify operational issues or security attacks (e.g., repeated failed login attempts).

## 4. Critical Business Logic (The "Secret Sauce")

### 1. Immutable Append-Only Storage
The service enforces an **Append-Only** pattern. There are **no Update or Delete** methods exposed in the repository. Once a log is written, it is historically frozen. This is critical for maintaining the integrity of the audit trail against tampering.

### 2. State Diffing (Old vs. New)
For update operations, the service captures both `OldValues` and `NewValues` in `JSONB` format.
-   **Logic**: This allows for precise reconstruction of what changed.
-   **Use Case**: If a merchant's status changes from `Active` to `Suspended`, the audit log captures the exact state of the merchant record before and after the change, proving *what* was modified.

### 3. Distributed Tracing Linkage
Every audit log includes `RequestID` and `CorrelationID`.
-   **Logic**: These IDs link the audit log entry to the specific API request and the broader distributed trace.
-   **Benefit**: Allows developers to correlate a specific database change (Audit Log) with the application logs and metrics (Prometheus/Grafana) for that specific request.

## 5. Database Schema

The service owns the `audit_logs` table.

| Column Name | Type | Description |
| :--- | :--- | :--- |
| `id` | `UUID` | Unique identifier for the log entry. |
| `actor_type` | `VARCHAR` | Type of actor (e.g., `system`, `merchant`, `admin`). |
| `actor_id` | `VARCHAR` | ID of the actor performing the action. |
| `action_category` | `VARCHAR` | High-level category (e.g., `payment`, `security`). |
| `resource_type` | `VARCHAR` | The type of entity affected (e.g., `transaction`, `user`). |
| `resource_id` | `UUID` | The ID of the specific entity affected. |
| `status` | `VARCHAR` | Outcome: `success`, `failed`, or `error`. |
| `old_values` | `JSONB` | Snapshot of data *before* the action. |
| `new_values` | `JSONB` | Snapshot of data *after* the action. |
| `created_at` | `TIMESTAMP` | **Immutable** timestamp of when the event occurred. |

## 6. Configuration & Env

The module relies on the global application configuration but specifically requires:

-   **Database Connection**: A valid `gorm.DB` instance connected to the primary PostgreSQL database.
-   **Logger**: A `logrus.Logger` instance for operational logging (distinct from the audit logs themselves).

### Dependencies
-   `gorm.io/gorm`: ORM for database interactions.
-   `github.com/lib/pq`: PostgreSQL driver (implied).
