#!/usr/local/bin/python

import jinja2
import os
import sys
import subprocess
import glob
import codecs
import time

# Utility functions
convert = lambda src, dst, environ: open(dst, "w").write(jinja2.Template(open(src).read()).render(**environ))

def check_arguments(environ, args):
    for argument in args:
        if argument not in environ:
            print("Environment variable %s is mandatory, exiting." % argument)
            sys.exit(2)

def generate_secrets(environ, secrets):
    for name, secret in secrets.items():
        if secret not in environ:
            filename = "/data/%s.%s.key" % (environ["SYNAPSE_SERVER_NAME"], name)
            if os.path.exists(filename):
                with open(filename) as handle: value = handle.read()
            else:
                print("Generating a random secret for {}".format(name))
                value = codecs.encode(os.urandom(32), "hex").decode()
                with open(filename, "w") as handle: handle.write(value)
            environ[secret] = value

def generate_tls_key_cert():
    servername = environ["SYNAPSE_SERVER_NAME"]
    subprocess.check_output([
        'openssl', 'req', '-x509', '-newkey', 'rsa:4096',
        '-keyout', f'/data/{servername}.tls.key',
        '-out', f'/data/{servername}.tls.crt',
        '-days', '365', '-nodes', '-subj', '/O=matrix',
    ])

# Prepare the configuration
mode = sys.argv[1] if len(sys.argv) > 1 else None
environ = os.environ.copy()

for e in environ:
    print("%s:%s" % (e, environ[e]))

ownership = "{}:{}".format(environ.get("UID", 991), environ.get("GID", 991))
args = ["python", "-m", "synapse.app.homeserver"]

# For now, always generate certificates, even if they are not used if TLS is disabled
generate_tls_key_cert()

# In generate mode, generate a configuration, missing keys, then exit
if mode == "generate":
    check_arguments(environ, ("SYNAPSE_SERVER_NAME", "SYNAPSE_REPORT_STATS", "SYNAPSE_CONFIG_PATH"))
    args += [
        "--server-name", environ["SYNAPSE_SERVER_NAME"],
        "--report-stats", environ["SYNAPSE_REPORT_STATS"],
        "--config-path", environ["SYNAPSE_CONFIG_PATH"],
        "--generate-config"
    ]
    os.execv("/usr/local/bin/python", args)

# In normal mode, generate missing keys if any, then run synapse
else:
    # Parse the configuration file
    if "SYNAPSE_CONFIG_PATH" in environ:
        args += ["--config-path", environ["SYNAPSE_CONFIG_PATH"]]
    else:
        check_arguments(environ, ("SYNAPSE_SERVER_NAME", "SYNAPSE_REPORT_STATS"))
        generate_secrets(environ, {
            "registration": "SYNAPSE_REGISTRATION_SHARED_SECRET",
            "macaroon": "SYNAPSE_MACAROON_SECRET_KEY"
        })
        environ["SYNAPSE_APPSERVICES"] = glob.glob("/data/appservices/*.yaml")
        if not os.path.exists("/compiled"): os.mkdir("/compiled")
        convert("/conf/homeserver.yaml", "/compiled/homeserver.yaml", environ)
        convert("/conf/log.config", "/compiled/log.config", environ)
        subprocess.check_output(["chown", "-R", ownership, "/data"])
        args += ["--config-path", "/compiled/homeserver.yaml"]
    # Generate missing keys and start synapse
    subprocess.check_output(args + ["--generate-keys"])

    # Initialise database
    for f in glob.glob("/usr/local/lib/python3.8/site-packages/synapse/storage/schema/*/full_schemas/72/full.sql.sqlite"):
        os.system(f"sqlite3 /data/homeserver.db < {f}")
    os.system("sqlite3 /data/homeserver.db < /usr/local/lib/python3.8/site-packages/synapse/storage/schema/common/schema_version.sql")
    os.system("echo \"insert into schema_version(lock, version, upgraded) values ('X', 72, false)\" | sqlite3 /data/homeserver.db")

    # Register test account
    sql = f"""insert into users(name, password_hash) values ('@matthew:{environ["SYNAPSE_SERVER_NAME"]}', '\\$2b\\$12\\$oOZr9g6bPScmPrpJHv/uuu2piCg7kN8ia/BAlfW6wske/1kLf8kze');
insert into access_tokens(id, user_id, token) values (123123, '@matthew:{environ["SYNAPSE_SERVER_NAME"]}', 'fake_token');"""
    os.system(f"echo \"{sql}\" | sqlite3 /data/homeserver.db");

    os.execv("/usr/local/bin/python", args)

