# indieweb-glue

A service that presents IndieWeb data in a non-CORS-restricted manner for embedding.

The `master` branch code is [running](https://indieweb-glue.evgenykuznetsov.org/) as a public service.

## API

`/api/hcard?url=URL` returns a JSON containing some information found in the [representative h-card](http://microformats.org/wiki/representative-h-card-parsing) on the page referenced by URL (if indeed there is a representative h-card).

`/api/photo?url=URL` returns the file referenced in the `u-photo` property of the abovementioned h-card.

`/api/pageinfo?url=URL` returns a JSON containing some information about the page referenced by URL.

`/api/opengraph?url=URL` returns a JSON containing some (currently very minimal) information from the [OpenGraph metadata](https://ogp.me/) that the page referenced by URL contains.

## Self-hosting

`go build` and run on your own server, if you wish. Settings are controlled through environment variables:

- `$URL` - the URL of the instance, defaults to `https://indieweb-glue.evgenykuznetsov.org`,
- `$PORT` - the port to run on, defaults to `8080`,
- `$MEMCACHIER_SERVERS`, `$MEMCACHIER_USERNAME`, `$MEMCACHIER_PASSWORD` - credentials to use `memcached`; if not supplied, the in-memory cache is used.