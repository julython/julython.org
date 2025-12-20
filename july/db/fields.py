"""
# Common Field Types

These represent complex reusable field types for postgres. This helps to keep the model definition simple like:

```python
from sqlmodel import Field, SQLModel

from .types import PrimaryKey, CreatedAt, UpdatedAt, JsonbData

class MyModel(SQLModel):
    id: uuid.UUID = PrimaryKey
    created_at: datetime = CreatedAt
    updated_at: datetime = UpdatedAt
    name: str = Field(min_length=1, max_length=100)
    description: str | None = None
    meta: dict[str, Any] = JsonbData(nullable=False)
```

**Note:** These require Python 3.14 for uuid7

## Database Types Exposed

### Core Entity Fields
- **PrimaryKey**: Auto-generated UUID field marked as primary key
- **CreatedAt**: Timestamp field with automatic creation time, indexed for performance
- **UpdatedAt**: Timestamp field with automatic updates on record changes, indexed for performance

### String Fields
- **ShortString**: Required string field with 100 character limit (VARCHAR(100))
- **LongText**: Required unlimited text field (PostgreSQL TEXT)
- **Email**: Optional email address field with 320 character limit (VARCHAR(320), RFC 5321 compliant)

### PostgreSQL-Specific Types
- **JsonbData**: JSONB field for structured data with efficient querying and indexing
- **StringArray**: Array of strings using PostgreSQL ARRAY(VARCHAR) type
- **IntArray**: Array of integers using PostgreSQL ARRAY(INTEGER) type
- **IpAddress**: Required IP address field using PostgreSQL INET type, indexed

"""

import uuid

from sqlalchemy import Integer, String, Text
from sqlalchemy.dialects.postgresql import (
    ARRAY,
    INET,
    JSONB,
    TIMESTAMP,
    UUID as SqlUUID,
)
from sqlmodel import Field

from july.utils import times


PrimaryKey = Field(
    default_factory=uuid.uuid7,  # type: ignore
    sa_type=SqlUUID,
    primary_key=True,
    description="Primary key identifier",
)


CreatedAt = Field(
    default_factory=times.now,
    sa_type=TIMESTAMP(timezone=True),
    nullable=False,
    index=True,
    description="Creation timestamp",
)

UpdatedAt = Field(
    default_factory=times.now,
    sa_type=TIMESTAMP(timezone=True),
    nullable=False,
    sa_column_kwargs={"onupdate": times.now},
    index=True,
    description="Last update timestamp",
)

ID = lambda nullable=True, index=False, unique=False, **kwargs: Field(
    sa_type=SqlUUID,
    nullable=nullable,
    index=index,
    unique=unique,
    **kwargs,
)

FK = lambda fk, nullable=False, index=True, unique=False, **kwargs: Field(
    foreign_key=fk,
    nullable=nullable,
    index=index,
    unique=unique,
    **kwargs,
)

Identifier = lambda nullable=True, index=False, unique=False, **kwargs: Field(
    max_length=255,
    sa_type=String(255),
    nullable=nullable,
    index=index,
    unique=unique,
    **kwargs,
)

ShortString = lambda length=100, nullable=True, index=False, **kwargs: Field(
    max_length=length,
    sa_type=String(length),
    nullable=nullable,
    index=index,
    **kwargs,
)


LongText = lambda nullable=True, **kwargs: Field(
    sa_type=Text,
    nullable=nullable,
    **kwargs,
)


Email = lambda nullable=True, description="Optional email address field (RFC 5321 compliant)", **kwargs: Field(
    max_length=320,
    sa_type=String(320),
    nullable=nullable,
    description=description,
    **kwargs,
)


# PostgreSQL specific types
Timestamp = lambda nullable=True, **kwargs: Field(
    sa_type=TIMESTAMP(timezone=True),
    nullable=nullable,
    **kwargs,
)

JsonbData = lambda nullable=True, **kwargs: Field(
    default_factory=dict,
    sa_type=JSONB,
    nullable=nullable,
    **kwargs,
)


Array = lambda nullable=True, type=String, **kwargs: Field(
    default_factory=list,
    sa_type=ARRAY(type),
    nullable=nullable,
    **kwargs,
)


IpAddress = lambda nullable=True, **kwargs: Field(
    sa_type=INET,
    index=True,
    nullable=nullable,
    **kwargs,
)
