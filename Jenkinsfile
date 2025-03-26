node {

    stage('Clone repository') {
        checkout scm
    }

    stage('Semantic Release') {
            withCredentials([usernamePassword(credentialsId: 'github-pat', usernameVariable: 'GITHUB_USERNAME', passwordVariable: 'GITHUB_TOKEN')]) {
                script {
                    echo "Running semantic-release"
                    sh "npx semantic-release"
                }
            }    
        }
    stage('Build and Push multi-platform image') {
        withCredentials([usernamePassword(credentialsId: 'docker-pat', usernameVariable: 'DOCKER_USERNAME', passwordVariable: 'DOCKER_TOKEN')]) {
            script{
                // Get the latest version
                def LATEST_TAG = sh(script: "git describe --tags --abbrev=0", returnStdout: true).trim()
                // Login to Docker
                sh """
                    docker login -u ${DOCKER_USERNAME} -p ${DOCKER_TOKEN}
                """
                //check make version
                sh """
                    make --version
                """
                
                // Build and push multi-platform image
                sh """
                    make docker-buildx IMG=roarceus/db-backup-operator:${LATEST_TAG}
                """
            }
        }
    }
}