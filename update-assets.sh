#!/bin/bash

# I am not a huge fan of frontend. I am even less of a fan of the ever changing
# eco-systems of different tools they have. I can't and at this point don't want
# to keep up. I also don't want to check in a boatload of minified css and js,
# so instead you're getting this simple bash script that will download them for you.
# TODO: run this from CI and check the actual assets for any missing links
# as of course new javascript and css should be in here.

mkdir -p assets/js assets/css

JQUERY_URL="https://code.jquery.com/jquery-3.6.0.min.js"
DATATABLES_URL="https://cdn.datatables.net/1.11.2/js/jquery.dataTables.min.js"

wget -O assets/js/jquery.min.js $JQUERY_URL
wget -O assets/js/jquery.dataTables.min.js $DATATABLES_URL

BOOTSTRAP_CSS_URL="https://cdn.jsdelivr.net/npm/bootstrap@5.1.1/dist/css/bootstrap.min.css"
BOOTSTRAP_JS_URL="https://cdn.jsdelivr.net/npm/bootstrap@5.1.1/dist/js/bootstrap.bundle.min.js"

wget -O assets/css/bootstrap.min.css $BOOTSTRAP_CSS_URL
wget -O assets/js/bootstrap.bundle.min.js $BOOTSTRAP_JS_URL