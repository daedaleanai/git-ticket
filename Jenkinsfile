def inDocker(body) {
    version = "v1.1.3"

    docker.withRegistry('https://gitea.daedalean.ai', 'jenkins_gitea_docker') {
        docker.image("gitea.daedalean.ai/jenkins/prod-builder-base:${version}").inside() {
            body()
        }
    }
}

pipeline {
    agent none

    options {
        timeout(time: 30, unit: 'MINUTES')
        timestamps()
        buildDiscarder(logRotator(daysToKeepStr: '7'))
    }

    stages {
        stage('Build') {
            agent {
                label "prod-docker-builder"
            }
            steps {
                echo 'Checking Web UI build'
                inDocker() {
                    sh '''
                    go generate

                    git diff --quiet || (echo "The generated web-ui assets are out of date. Please run 'go generate'" && false)
                    echo "web-ui assets are up to date"
                    '''
                }
            }
        }
        stage('Tests') {
            agent {
                label "prod-docker-builder"
            }
            steps {
                script {
                    inDocker() {
                        sh '''
                            go install github.com/jstemmer/go-junit-report@latest
                            go install github.com/t-yuki/gocover-cobertura@latest
                            rm -f report.xml
                            go test -v -covermode=atomic -coverprofile=coverage.out -bench=. ./... | go-junit-report -set-exit-code > report.xml
                            gocover-cobertura < coverage.out > coverage.xml
                        '''
                    }
                }
            }
            post {
                always{
                    junit 'report.xml'
                    publishCoverage(
                        adapters: [
                            istanbulCobertura(path: 'coverage.xml')
                        ],
                        sourceFileResolver: sourceFiles('STORE_ALL_BUILD'),
                        failNoReports: true,
                        globalThresholds: [
                            [
                                thresholdTarget: 'Line',
                            ]
                        ]
                    )
                }
                cleanup {
                    cleanWs(disableDeferredWipeout: true, notFailBuild: true)
                }
            }
        }
    }
}
