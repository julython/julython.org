from july.utils import times


def test_from_now():
    current = times.now()

    future = times.from_now(days=4)

    assert future > current


def test_parse_timestamp():
    parsed = times.parse_timestamp("2024-07-04")
    assert parsed.year == 2024
    assert parsed.month == 7

    timestamp = times.parse_timestamp(1334023345)
    assert timestamp.year == 2012
