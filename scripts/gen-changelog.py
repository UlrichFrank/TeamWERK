#!/usr/bin/env python3
import sys
import re

groups: dict[str, list[str]] = {}
order: list[str] = []

for line in sys.stdin:
    line = line.rstrip('\n')
    m = re.match(r'^([^|]+)\|(feat|fix)(\(([^)]*)\))?:(.+)$', line)
    if not m:
        continue
    date = m.group(1)
    type_ = m.group(2)
    scope = m.group(4) or '?'
    message = m.group(5).strip()
    if date not in groups:
        order.append(date)
        groups[date] = []
    groups[date].append(f'- [{type_}] {scope}: {message}')

print('\n\n'.join(f'## {d}\n' + '\n'.join(groups[d]) for d in order))
