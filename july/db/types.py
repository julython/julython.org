"""
These represent complex reusable field types using `Annotated`. This helps to keep the model definition simple like:

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
    default_factory=uuid.uuid7,
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

ID = lambda nullable=True, description="UUID", index=False, unique=False: Field(
    sa_type=SqlUUID,
    nullable=nullable,
    index=index,
    unique=unique,
    description=description,
)

FK = lambda fk, nullable=False, description="FK", index=True, unique=False: Field(
    foreign_key=fk,
    nullable=nullable,
    index=index,
    unique=unique,
    description=description,
)

Identifier = (
    lambda nullable=True, description="Identifier", index=False, unique=False: Field(
        max_length=255,
        sa_type=String(255),
        nullable=nullable,
        index=index,
        unique=unique,
        description=description,
    )
)

ShortString = lambda length=100, nullable=True, index=False, description="Short text field": Field(
    max_length=length,
    sa_type=String(length),
    nullable=nullable,
    index=index,
    description=description,
)


LongText = lambda nullable=True, description="Long text field": Field(
    sa_type=Text,
    nullable=nullable,
    description=description,
)


Email = lambda nullable=True, description="Optional email address field (RFC 5321 compliant)": Field(
    max_length=320,
    sa_type=String(320),
    nullable=nullable,
    description=description,
)


# PostgreSQL specific types
Timestamp = lambda nullable=True, description="Datetime field": Field(
    sa_type=TIMESTAMP(timezone=True),
    nullable=nullable,
    description=description,
)

JsonbData = lambda nullable=True, description="JSONB field": Field(
    default_factory=dict,
    sa_type=JSONB,
    nullable=nullable,
    description=description,
)


StringArray = lambda nullable=True, description="Array of strings": Field(
    default_factory=list,
    sa_type=ARRAY(String),
    nullable=nullable,
    description=description,
)


IntArray = lambda nullable=True, description="Array of integers": Field(
    default_factory=list,
    sa_type=ARRAY(Integer),
    nullable=nullable,
    description=description,
)


IpAddress = lambda nullable=True, description="IP address": Field(
    sa_type=INET,
    index=True,
    nullable=nullable,
    description=description,
)
