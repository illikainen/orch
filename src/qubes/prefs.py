#!/usr/bin/env python3

import argparse
import json
import sys

import qubesadmin


def cmd_get(args):
    qubes = qubesadmin.Qubes()
    return get_prefs(qubes.domains[args.name])


def get_prefs(domain):
    prefs = {
        "defaults": [],
    }
    for key in domain.property_list():
        try:
            prefs[to_json(key)] = getattr(domain, key)
            if isinstance(prefs[to_json(key)], (qubesadmin.label.Label, qubesadmin.vm.QubesVM)):
                prefs[to_json(key)] = prefs[key].name

            if domain.property_is_default(key):
                prefs["defaults"].append(to_json(key))
        except AttributeError:
            pass

    return prefs


def cmd_set(args):
    qubes = qubesadmin.Qubes()
    prefs = json.loads(sys.stdin.read())
    return set_prefs(qubes.domains[args.name], prefs, args.dry_run)


def set_prefs(domain, prefs, dry_run):
    old = get_prefs(domain)
    changes = []

    for key, value in prefs.items():
        if key == "defaults":
            for default_key in value:
                if not domain.property_is_default(from_json(default_key)):
                    old_value = old.get(default_key)
                    default_value = domain.property_get_default(from_json(default_key))
                    changes.append({
                        "property": default_key,
                        "old_value": str(old_value),
                        "old_is_default": False,
                        "new_value": str(default_value),
                        "default": True,
                    })
                    if not dry_run:
                        delattr(domain, from_json(default_key))
        else:
            old_value = old.get(key)
            if value != old_value and not (value == "" and old_value == None):
                changes.append({
                    "property": key,
                    "old_value": str(old_value) if old_value else "*unset*",
                    "old_is_default": domain.property_is_default(key),
                    "new_value": str(value) if value else "*unset*",
                    "new_is_default": False,
                })
                if not dry_run:
                    setattr(domain, from_json(key), value)

    return changes


def to_json(key):
    if key == "klass":
        return "class"
    return key


def from_json(key):
    if key == "class":
        return "klass"
    return key


def parse_args():
    ap = argparse.ArgumentParser()
    ap.set_defaults(fn=lambda _: sys.exit(ap.format_usage().strip()))

    sp = ap.add_subparsers()

    get = sp.add_parser("get")
    get.add_argument("name")
    get.set_defaults(fn=cmd_get)

    set_ = sp.add_parser("set")
    set_.add_argument("--dry-run", action="store_true")
    set_.add_argument("name")
    set_.set_defaults(fn=cmd_set)

    return ap.parse_args()


def main():
    args = parse_args()
    print(json.dumps(args.fn(args), indent=4))


if __name__ == "__main__":
    main()
