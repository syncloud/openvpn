local name = 'openvpn';
local browser = 'firefox';
local go = '1.25';
local nginx = '1.29.3-alpine3.22';
local debian = 'bookworm-slim';
local openvpn = '2.6.16';
local easyrsa = '3.2.4';
local gcc = '13.4.0';
local openssl = '3.1.4';
local platform = '25.09';
local selenium = '4.35.0-20250828';
local deployer = 'https://github.com/syncloud/store/releases/download/4/syncloud-release';
local python = '3.12-slim-bookworm';
local distro_default = 'bookworm';

local build(arch, test_ui, dind) = [
  {
    kind: 'pipeline',
    type: 'docker',
    name: arch,
    platform: {
      os: 'linux',
      arch: arch,
    },
    steps: [
             {
               name: 'version',
               image: 'debian:' + debian,
               commands: [
                 'echo $DRONE_BUILD_NUMBER > version',
               ],
             },
             {
               name: 'download',
               image: 'debian:' + debian,
               commands: [
                 './download.sh ',
               ],
             },
             {
               name: 'openssl',
               image: 'alpine/openssl:' + openssl,
               commands: [
                 './openssl/build.sh',
               ],
             },
             {
               name: 'openssl test',
               image: 'syncloud/platform-' + distro_default + '-' + arch + ':' + platform,
               commands: [
                 './openssl/test.sh',
               ],
             },
             {
               name: 'nginx',
               image: 'nginx:' + nginx,
               commands: [
                 './nginx/build.sh',
               ],
             },
             {
               name: 'nginx test',
               image: 'syncloud/platform-' + distro_default + '-' + arch + ':' + platform,
               commands: [
                 './nginx/test.sh',
               ],
             },
             {
               name: 'openvpn',
               image: 'gcc:' + gcc,
               commands: [
                 './openvpn/build.sh ' + openvpn + ' ' + easyrsa,
               ],
             },
             {
               name: 'openvpn test',
               image: 'syncloud/platform-' + distro_default + '-' + arch + ':' + platform,
               commands: [
                 './openvpn/test.sh',
               ],
             },
             {
               name: 'build web',
               image: 'golang:' + go,
               commands: [
                 'cd web',
                 './build.sh',
               ],
             },

             {
               name: 'package python',
               image: 'docker:' + dind,
               commands: [
                 './python/build.sh',
               ],
               volumes: [
                 {
                   name: 'dockersock',
                   path: '/var/run',
                 },
               ],
             },
             {
               name: 'package',
               image: 'debian:' + debian,
               commands: [
                 'VERSION=$(cat version)',
                 './package.sh ' + name + ' $VERSION ',
               ],
             },
             {
               name: 'test',
               image: 'python:' + python,
               commands: [
                 'cd test',
                 './deps.sh',
                 'py.test -x -s test.py --distro=' + distro_default + ' --ver=$DRONE_BUILD_NUMBER --app=' + name,
               ],
             },
           ] +
           (if test_ui then ([
                               {
                                 name: 'selenium',
                                 image: 'selenium/standalone-' + browser + ':' + selenium,
                                 detach: true,
                                 environment: {
                                   SE_NODE_SESSION_TIMEOUT: '999999',
                                   START_XVFB: 'true',
                                 },
                                 volumes: [{
                                   name: 'shm',
                                   path: '/dev/shm',
                                 }],
                                 commands: [
                                   'cat /etc/hosts',
                                   'DOMAIN="' + distro_default + '.com"',
                                   'APP_DOMAIN="' + name + '.' + distro_default + '.com"',
                                   'getent hosts $APP_DOMAIN | sed "s/$APP_DOMAIN/auth.$DOMAIN/g" | sudo tee -a /etc/hosts',
                                   'cat /etc/hosts',
                                   '/opt/bin/entry_point.sh',
                                 ],
                               },
                               {
                                 name: 'selenium-video',
                                 image: 'selenium/video:ffmpeg-4.3.1-20220208',
                                 detach: true,
                                 environment: {
                                   DISPLAY_CONTAINER_NAME: 'selenium',
                                   PRESET: '-preset ultrafast -movflags faststart',
                                 },
                                 volumes: [
                                   {
                                     name: 'shm',
                                     path: '/dev/shm',
                                   },
                                   {
                                     name: 'videos',
                                     path: '/videos',
                                   },
                                 ],
                               },
                               {
                                 name: 'test-ui',
                                 image: 'python:' + python,
                                 commands: [
                                   'cd test',
                                   './deps.sh',
                                   'py.test -x -s ui.py --distro=' + distro_default + ' --ver=$DRONE_BUILD_NUMBER --app=' + name + ' --browser=' + browser,
                                 ],
                                 volumes: [{
                                   name: 'videos',
                                   path: '/videos',
                                 }],
                               },
                               {
                                 name: 'test-upgrade',
                                 image: 'python:' + python,
                                 commands: [
                                   'cd test',
                                   './deps.sh',
                                   'py.test -x -s upgrade.py --distro=' + distro_default + ' --ver=$DRONE_BUILD_NUMBER --app=' + name + ' --browser=' + browser,
                                 ],
                                 privileged: true,
                                 volumes: [{
                                   name: 'videos',
                                   path: '/videos',
                                 }],
                               },
                             ]) else []) + [
      {
        name: 'upload',
        image: 'debian:' + debian,
        environment: {
          AWS_ACCESS_KEY_ID: {
            from_secret: 'AWS_ACCESS_KEY_ID',
          },
          AWS_SECRET_ACCESS_KEY: {
            from_secret: 'AWS_SECRET_ACCESS_KEY',
          },
          SYNCLOUD_TOKEN: {
            from_secret: 'SYNCLOUD_TOKEN',
          },
        },
        commands: [
          'PACKAGE=$(cat package.name)',
          'apt update && apt install -y wget',
          'wget ' + deployer + '-' + arch + ' -O release --progress=dot:giga',
          'chmod +x release',
          './release publish -f $PACKAGE -b $DRONE_BRANCH',
        ],
        when: {
          branch: ['stable', 'master'],
          event: ['push'],
        },
      },
      {
        name: 'promote',
        image: 'debian:' + debian,
        environment: {
          AWS_ACCESS_KEY_ID: {
            from_secret: 'AWS_ACCESS_KEY_ID',
          },
          AWS_SECRET_ACCESS_KEY: {
            from_secret: 'AWS_SECRET_ACCESS_KEY',
          },
          SYNCLOUD_TOKEN: {
            from_secret: 'SYNCLOUD_TOKEN',
          },
        },
        commands: [
          'apt update && apt install -y wget',
          'wget ' + deployer + '-' + arch + ' -O release --progress=dot:giga',
          'chmod +x release',
          './release promote -n ' + name + ' -a $(dpkg --print-architecture)',
        ],
        when: {
          branch: ['stable'],
          event: ['push'],
        },
      },
      {
        name: 'artifact',
        image: 'appleboy/drone-scp:1.6.4',
        settings: {
          host: {
            from_secret: 'artifact_host',
          },
          username: 'artifact',
          key: {
            from_secret: 'artifact_key',
          },
          timeout: '2m',
          command_timeout: '2m',
          target: '/home/artifact/repo/' + name + '/${DRONE_BUILD_NUMBER}-' + arch,
          source: 'artifact/*',
          strip_components: 1,
        },
        when: {
          status: ['failure', 'success'],
          event: ['push'],
        },
      },
    ],
    trigger: {
      event: [
        'push',
        'pull_request',
      ],
    },
    services: [
      {
        name: 'docker',
        image: 'docker:' + dind,
        privileged: true,
        volumes: [
          {
            name: 'dockersock',
            path: '/var/run',
          },
        ],
      },
      {
        name: name + '.' + distro_default + '.com',
        image: 'syncloud/platform-' + distro_default + '-' + arch + ':' + platform,
        privileged: true,
        volumes: [
          {
            name: 'dbus',
            path: '/var/run/dbus',
          },
          {
            name: 'dev',
            path: '/dev',
          },
        ],
      },
    ],
    volumes: [
      {
        name: 'dbus',
        host: {
          path: '/var/run/dbus',
        },
      },
      {
        name: 'dev',
        host: {
          path: '/dev',
        },
      },
      {
        name: 'shm',
        temp: {},
      },
      {
        name: 'videos',
        temp: {},
      },
      {
        name: 'dockersock',
        temp: {},
      },
    ],
  },
  {
    kind: 'pipeline',
    type: 'docker',
    name: 'promote-' + arch,
    platform: {
      os: 'linux',
      arch: arch,
    },
    steps: [
      {
        name: 'promote',
        image: 'debian:' + debian,
        environment: {
          AWS_ACCESS_KEY_ID: {
            from_secret: 'AWS_ACCESS_KEY_ID',
          },
          AWS_SECRET_ACCESS_KEY: {
            from_secret: 'AWS_SECRET_ACCESS_KEY',
          },
        },
        commands: [
          'apt update && apt install -y wget',
          'wget https://github.com/syncloud/snapd/releases/download/1/syncloud-release-' + arch + ' -O release --progress=dot:giga',
          'chmod +x release',
          './release promote -n ' + name + ' -a $(dpkg --print-architecture)',
        ],
      },
    ],
    trigger: {
      event: [
        'promote',
      ],
    },
  },
];

build('amd64', true, '20.10.21-dind') +
build('arm64', false, '20.10.21-dind') +
build('arm', false, '19.03.8-dind')
