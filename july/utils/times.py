from datetime import datetime, timezone, timedelta


def now() -> datetime:
    """Returns the current datetime in UTC"""
    return datetime.now(timezone.utc)


def from_now(**kwargs) -> datetime:
    """Return the time in days,hours,minutes from now.

    ```python
    yesterday = from_now(days=-1)
    ```

    Passes all the key word args directly to `timedelta`
    """
    currently = now()
    return currently + timedelta(**kwargs)


def parse_timestamp(ts: str) -> datetime:
    ts = ts.replace("Z", "+00:00")
    return datetime.fromisoformat(ts)
