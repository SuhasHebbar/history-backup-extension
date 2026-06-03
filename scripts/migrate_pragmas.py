#!/usr/bin/env python3
"""
One-time migration script to apply initPragmas to an existing SQLite database.

Some pragmas (page_size, auto_vacuum) cannot be changed once tables exist without
first disabling WAL and running VACUUM. Run this while the Go server is stopped.

Usage:
    python migrate_pragmas.py <path-to-history.db>
"""

import argparse
import sqlite3


def migrate(db_path: str) -> None:
    con = sqlite3.connect(db_path)
    try:
        cur = con.cursor()

        # Disable WAL — required before VACUUM can change page_size.
        cur.execute("PRAGMA journal_mode = DELETE")

        cur.execute("PRAGMA synchronous = NORMAL")
        cur.execute("PRAGMA busy_timeout = 5000")
        cur.execute("PRAGMA cache_size = -20000")
        cur.execute("PRAGMA foreign_keys = ON")
        cur.execute("PRAGMA auto_vacuum = INCREMENTAL")
        cur.execute("PRAGMA temp_store = MEMORY")
        cur.execute("PRAGMA mmap_size = 2147483648")
        cur.execute("PRAGMA page_size = 8192")

        # Rebuild the database — applies page_size and auto_vacuum mode changes.
        cur.execute("VACUUM")

        cur.execute("PRAGMA journal_mode = WAL")

        print("Migration complete. Final pragma values:")
        for pragma in (
            "journal_mode",
            "page_size",
            "auto_vacuum",
            "synchronous",
            "busy_timeout",
            "cache_size",
            "foreign_keys",
            "temp_store",
            "mmap_size",
        ):
            cur.execute(f"PRAGMA {pragma}")
            row = cur.fetchone()
            print(f"  {pragma} = {row[0] if row else 'n/a'}")
    finally:
        con.close()


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("db_path", help="Path to the SQLite database file")
    args = parser.parse_args()
    migrate(args.db_path)


if __name__ == "__main__":
    main()
