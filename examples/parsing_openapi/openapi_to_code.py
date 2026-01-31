"""Lightweight accessor for x-db metadata in OpenAPI schema."""

from __future__ import annotations

import json
from pathlib import Path
from typing import Any, Iterator
from dataclasses import dataclass, field


# --- Data models ---


@dataclass
class Column:
    """A single database column parsed from an OpenAPI property + its x-db extension."""

    name: str
    type: str
    sql_type: dict[str, str] = field(default_factory=dict)
    format: str | None = None
    nullable: bool = False
    read_only: bool = False
    write_only: bool = False
    default: Any = None
    semantic: str | None = None
    generated: bool = False
    auto_managed: bool | None = None
    enum: list[str] | None = None
    max_length: int | None = None
    min_length: int | None = None
    pattern: str | None = None
    fk: str | None = None
    ref: str | None = None
    relation: str | None = None
    on_delete: str | None = None
    on_update: str | None = None
    inverse_of: str | None = None
    virtual: bool = False
    storage: str | None = None
    computed: dict | None = None

    @property
    def pg_type(self) -> str:
        return self.sql_type.get("postgres", "")

    @property
    def sqlite_type(self) -> str:
        return self.sql_type.get("sqlite", "")

    @property
    def is_computed(self) -> bool:
        return self.computed is not None

    @property
    def is_app_only(self) -> bool:
        return self.storage == "app_only"


@dataclass
class Index:
    """A database index."""

    name: str
    columns: list[str]
    unique: bool = False
    primary: bool = False


@dataclass
class Relationship:
    """A relationship between two tables (has_many, has_one, or many_to_many)."""

    name: str
    type: str
    target: str
    target_table: str
    local_key: str | None = None
    foreign_key: str | None = None
    backref: str | None = None
    through: dict[str, str] | None = None


@dataclass
class Table:
    """A database table parsed from an OpenAPI schema + its x-db extension."""

    schema_name: str
    table: str
    namespace: str
    columns: dict[str, Column] = field(default_factory=dict)
    indexes: list[Index] = field(default_factory=list)
    relationships: dict[str, Relationship] = field(default_factory=dict)
    primary_key: list[str] = field(default_factory=list)
    timestamps: bool = False
    soft_delete: bool = False
    searchable: list[str] = field(default_factory=list)
    join_table: dict[str, Any] | None = None

    def __getitem__(self, col: str) -> Column:
        return self.columns[col]

    @property
    def pk(self) -> list[str]:
        return self.primary_key

    @property
    def fk_columns(self) -> dict[str, Column]:
        return {k: v for k, v in self.columns.items() if v.fk}

    @property
    def unique_columns(self) -> list[str]:
        return [
            idx.columns[0]
            for idx in self.indexes
            if idx.unique and len(idx.columns) == 1 and not idx.primary
        ]

    @property
    def computed_columns(self) -> dict[str, Column]:
        return {k: v for k, v in self.columns.items() if v.is_computed}

    @property
    def is_join_table(self) -> bool:
        """True if this table is an auto-generated many-to-many join table."""
        return self.join_table is not None


# --- Parser ---


def _parse_column(name: str, prop: dict[str, Any]) -> Column:
    """Turn one OpenAPI property definition into a Column."""
    xdb = prop.get("x-db", {})
    return Column(
        name=name,
        type=prop.get("type", ""),
        sql_type=xdb.get("sql_type", {}),
        format=prop.get("format"),
        nullable=prop.get("nullable", False),
        read_only=prop.get("readOnly", False),
        write_only=prop.get("writeOnly", False),
        default=xdb.get("default"),
        semantic=xdb.get("semantic"),
        generated=xdb.get("generated", False),
        auto_managed=xdb.get("auto_managed"),
        enum=prop.get("enum"),
        max_length=prop.get("maxLength"),
        min_length=prop.get("minLength"),
        pattern=prop.get("pattern"),
        fk=xdb.get("fk"),
        ref=xdb.get("ref"),
        relation=xdb.get("relation"),
        on_delete=xdb.get("on_delete"),
        on_update=xdb.get("on_update"),
        inverse_of=xdb.get("inverse_of"),
        virtual=xdb.get("virtual", False),
        storage=xdb.get("storage"),
        computed=xdb.get("computed"),
    )


def _parse_index(raw: dict[str, Any]) -> Index:
    """Turn one index definition into an Index."""
    return Index(
        name=raw["name"],
        columns=raw["columns"],
        unique=raw.get("unique", False),
        primary=raw.get("primary", False),
    )


def _parse_relationship(name: str, raw: dict[str, Any]) -> Relationship:
    """Turn one relationship definition into a Relationship."""
    return Relationship(
        name=name,
        type=raw["type"],
        target=raw["target"],
        target_table=raw.get("target_table", ""),
        local_key=raw.get("local_key"),
        foreign_key=raw.get("foreign_key"),
        backref=raw.get("backref"),
        through=raw.get("through"),
    )


def _parse_table(name: str, defn: dict[str, Any]) -> Table:
    """Turn one OpenAPI schema definition into a Table."""
    xdb = defn.get("x-db", {})
    properties = defn.get("properties", {})

    columns = {
        col_name: _parse_column(col_name, col_def)
        for col_name, col_def in properties.items()
    }
    indexes = [_parse_index(index_def) for index_def in xdb.get("indexes", [])]
    relationships = {
        rel_name: _parse_relationship(rel_name, rel_def)
        for rel_name, rel_def in xdb.get("relationships", {}).items()
    }

    return Table(
        schema_name=name,
        table=xdb.get("table", ""),
        namespace=xdb.get("namespace", ""),
        columns=columns,
        indexes=indexes,
        relationships=relationships,
        primary_key=xdb.get("primary_key", []),
        timestamps=xdb.get("timestamps", False),
        soft_delete=xdb.get("soft_delete", False),
        searchable=xdb.get("searchable", []),
        join_table=xdb.get("join_table"),
    )


class Schema:
    """Load an OpenAPI spec and expose x-db metadata as typed objects."""

    def __init__(self, path: str | Path = "openapi.json") -> None:
        raw: dict[str, Any] = json.loads(Path(path).read_text())
        self.info: dict[str, Any] = raw.get("info", {})

        schemas: dict[str, Any] = raw.get("components", {}).get("schemas", {})
        self.tables: dict[str, Table] = {
            name: _parse_table(name, defn) for name, defn in schemas.items()
        }

    def __getitem__(self, key: str) -> Table:
        return self.tables[key]

    def __iter__(self) -> Iterator[Table]:
        return iter(self.tables.values())

    def by_table(self, table_name: str) -> Table | None:
        """Find a table by its SQL table name (e.g. ``"blog_post"``)."""
        return next((table for table in self if table.table == table_name), None)

    def by_namespace(self, namespace: str) -> list[Table]:
        """Return all tables in a namespace (e.g. ``"auth"``)."""
        return [table for table in self if table.namespace == namespace]
