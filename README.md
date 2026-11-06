![Verify Build](https://github.com/Apicurio/apicurito/workflows/Verify%20Build/badge.svg)

# Apicurito (Apicurio Lite)

This project was generated with [Angular CLI](https://github.com/angular/angular-cli) version 7.3.3.

## Development server

Before running the dev server you must install `node` and `yarn` (Node.js 16.x is the tested
version, see [Building the Image](#building-the-image)). Once these tools are installed
you must run `yarn install` to download all of the project's dependencies from `ui` folder.

Run `ng serve` from `ui` folder for a dev server. Navigate to `http://localhost:4200/`. The app will automatically 
reload if you change any of the source files.

To run on a different port, try `ng serve --port 8888` for example. And if you want AOT enabled, 
you can run `ng serve --aot --port 8888`.

### Testing Service Worker

The service worker is disabled for development builds but can still be easily tested locally.

From the `ui` folder, first run `npm run build` or `ng build --prod` to build the production app.
Then serve the `dist` folder with any web server on a different port from the dev server, and with
caching disabled. The simplest way to do this is: `npx http-server -p 8080 -c-1 dist`

You can also check out the [Angular docs](https://angular.io/guide/service-worker-getting-started)
for more detailed docs.

## Building the Image

The full build is driven by maven, which downloads the pinned Node.js and Yarn toolchain
itself (no local Node.js install required):

    mvn clean install -P image

The toolchain is pinned in `ui/pom.xml` to Node.js `v16.20.2` and Yarn `1.13.0`. Node 16 is
the newest release compatible with the webpack 4 based build (webpack 4 relies on the OpenSSL
`md4` hash, which was removed in Node 17), and it is the first Node line with official
`darwin-arm64` builds, so the build runs natively on Apple Silicon as well as on x86_64.

The image is a single static Go binary (`ui/cmd/`) with the compiled Angular application
embedded at build time, running on a digest-pinned `alpine:3.24.3` base image (see
`ui/docker/Dockerfile`). There are no other runtime dependencies.

The same steps by hand, starting from the `ui` folder:

    yarn install
    yarn build
    cd cmd
    ./build.sh
    mkdir -p ../target/docker-image/maven/target/go/linux-amd64
    cp ../target/go/linux-amd64/apicurito-ui ../target/docker-image/maven/target/go/linux-amd64/
    cp -R ../docker/licenses ../target/docker-image/
    docker build -f ../docker/Dockerfile -t apicurito/apicurito-ui:local ../target/docker-image/
    docker run -p 8080:8080 apicurito/apicurito-ui:local

Then open http://localhost:8080

Note: the maven build additionally copies `ui/static/` into the dist folder (the `copy-static`
step in `ui/pom.xml`); the hand-built steps above skip that, matching the published image.

## Running On Openshift

This will try to pull down the Apicurito image from container registries:

    oc replace --force  -f openshift.yml

But if you want that configuration changed so that it uses local builds, run:

    ./openshift-dev.sh

And build the image using:

    mvn install -P image    

And if you have deployed the [fuse-apicurito-generator](https://github.com/jboss-fuse/fuse-apicurito-generator) 
into the current project, then run the following to configure Apicurito to use it.

	./enable-fuse-generator.sh 

## Publishing the Image

    mvn clean install -P image -Dfabric8.mode=kubernetes
    docker push apicurio/apicurito-ui:latest
