#!/bin/bash

# I am not a huge fan of frontend. I am even less of a fan of the ever changing
# eco-systems of different tools they have. I can't and at this point don't want
# to keep up. I also don't want to check in a boatload of minified css and js,
# so instead you're getting this simple bash script that will download them for you.
# TODO: run this from CI and check the actual assets for any missing links
# as of course new javascript and css should be in here.

mkdir -p assets/js/lib assets/css

JQUERY_URL="https://code.jquery.com/jquery-3.6.0.min.js"
DATATABLES_URL="https://cdn.datatables.net/1.11.3/js/jquery.dataTables.min.js"
DATATABLES_BOOTSTRAP_JS_URL="https://cdn.datatables.net/1.11.3/js/dataTables.bootstrap5.min.js"
DATATABLES_BOOTSTRAP_CSS_URL="https://cdn.datatables.net/1.11.3/css/dataTables.bootstrap5.min.css"

wget -O assets/js/lib/jquery.min.js $JQUERY_URL
wget -O assets/js/lib/jquery.dataTables.min.js $DATATABLES_URL
wget -O assets/js/lib/dataTables.bootstrap5.min.js $DATATABLES_BOOTSTRAP_JS_URL
wget -O assets/css/dataTables.bootstrap5.min.css $DATATABLES_BOOTSTRAP_CSS_URL

BOOTSTRAP_CSS_URL="https://cdn.jsdelivr.net/npm/bootstrap@5.1.3/dist/css/bootstrap.min.css"
BOOTSTRAP_JS_URL="https://cdn.jsdelivr.net/npm/bootstrap@5.1.3/dist/js/bootstrap.bundle.min.js"

wget -O assets/css/bootstrap.min.css $BOOTSTRAP_CSS_URL
wget -O assets/js/lib/bootstrap.bundle.min.js $BOOTSTRAP_JS_URL

XTERM_CSS_URL="https://cdn.jsdelivr.net/npm/xterm@latest/css/xterm.css"
XTERM_JS_URL="https://cdn.jsdelivr.net/npm/xterm@latest/lib/xterm.min.js"

wget -O assets/css/xterm.css $XTERM_CSS_URL
wget -O assets/js/lib/xterm.min.js $XTERM_JS_URL

TUS_JS_URL="https://cdn.jsdelivr.net/npm/tus-js-client@latest/dist/tus.min.js"

wget -O assets/js/lib/tus.min.js $TUS_JS_URL