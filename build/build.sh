#!/bin/bash
set -ex

mkdir -p {$RELEASEDIR/,$RELEASEDIR/build/,$RELEASEDIR/rpm/,$RELEASEDIR/deb/}
rm -rf {$RELEASEDIR/build/*,$RELEASEDIR/rpm/*,$RELEASEDIR/deb/*}
mkdir -p {$RELEASEDIR/build/opt/natlog/bin/,$RELEASEDIR/build/opt/natlog/etc/,$RELEASEDIR/build/opt/natlog/var/db/,$RELEASEDIR/deb/conf,$RELEASEDIR/build/$UNITDIR}

make build

cp ./bin/natlog $RELEASEDIR/build/opt/natlog/bin
cp ./contrib/natlog.yaml.example $RELEASEDIR/build/opt/natlog/etc
cp ./contrib/system.d/natlog.service $RELEASEDIR/build/$UNITDIR
cp ./contrib/debian-repo-config $RELEASEDIR/deb/conf/distributions
cp ./contrib/yumrepo.repo $RELEASEDIR/rpm/natlog.repo

touch $RELEASEDIR/build/opt/natlog/etc/natlog.yaml



# Build packages
fpm -s dir -t deb -n natlog --deb-user="nobody" --deb-group="nogroup" --license=MIT --architecture="amd64" --vendor="Alexander Tischenko <tsm@fiberside.ru>" --deb-systemd=$RELEASEDIR/build/$UNITDIR/natlog.service --config-files=/opt/natlog/etc --version ${VERSION/v/} --iteration $BUILD --depends libsqlite3-0 --description "YubiServ - Cloud-free YubiKey verification service." --before-install $BUILD_DIR/contrib/beforeinstall.sh --after-install $BUILD_DIR/contrib/afterinstall.sh --before-remove $BUILD_DIR/contrib/beforeremove.sh --after-upgrade $BUILD_DIR/contrib/afterupgrade.sh -p $RELEASEDIR/deb -C $RELEASEDIR/build .