# RBAC API

Base path: `/panel/api/rbac`

## Roles

### List Roles
```http
GET /panel/api/rbac/roles
```

### Get Role Permissions
```http
GET /panel/api/rbac/roles/:id/permissions
```

### Update Role Permissions
```http
PUT /panel/api/rbac/roles/:id/permissions
```

## Permissions

### List All Permissions
```http
GET /panel/api/rbac/permissions
```

## User Roles

### Get User Role
```http
GET /panel/api/rbac/users/:userId/role
```

### Assign User Role
```http
PUT /panel/api/rbac/users/:userId/role
```

## Default Roles

| Role | Description |
|------|-------------|
| `admin` | Full control over all resources |
| `operator` | Protocol management + monitoring |
| `viewer` | Read-only access |
| `service` | API-only access |

## Permissions Matrix

| Resource | Actions |
|----------|---------|
| `users` | read, write, delete |
| `inbounds` | read, write, delete |
| `protocols` | read, write, control |
| `services` | read, write, control |
| `wrappers` | read, write, control |
| `monitoring` | read, write |
| `settings` | read, write |
| `backup` | read, write |
| `audit` | read |
| `roles` | read, write |
| `certificates` | read, write |
