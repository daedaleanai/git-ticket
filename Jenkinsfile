pipeline {
    agent none

    options {
        timeout(time: 30, unit: 'MINUTES')
        timestamps()
        buildDiscarder(logRotator(daysToKeepStr: '7'))
    }

    stages {
        stage('Tests') {
            agent {
                label "vxs-build"
            }
            steps {
                script {
                    sh '''
                        go install github.com/jstemmer/go-junit-report@latest
                        go install github.com/t-yuki/gocover-cobertura@latest
                        rm -f report.xml
                        go test -v -covermode=atomic -coverprofile=coverage.out -bench=. ./... | go-junit-report -set-exit-code > report.xml
                        gocover-cobertura < coverage.out > coverage.xml
                    '''
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
                                unhealthyThreshold: 10.0,
                                unstableThreshold: 30.0
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