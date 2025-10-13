#!/bin/bash
set +e

execute_command() {
    local description=$1
    shift
    local command=("$@")

    echo ========================================================================
    
    echo "Executing: ${command[*]}"
    output=$(go run "${command[@]}" 2>&1)
    local exit_code=$?
    
    if [ $exit_code -ne 0 ]; then
        echo "Failed to $description"
        echo "$output"
        return 1
    fi
    
    echo "$description success"
    echo "$output"
    echo
    echo
    return 0
}

sls_project_test() {
    TIMESTAMP=$(date +"%Y%m%d-%H%M")
    PROJECT_NAME="mx-pro-$TIMESTAMP-$RANDOM"
    DESCRIPTION="this is test"

    echo "###### Try to test project $PROJECT_NAME ######"

    if ! execute_command "create project $PROJECT_NAME" ./main/main.go Sls CreateProject --body "{\"description\":\"$DESCRIPTION\",\"projectName\":\"$PROJECT_NAME\"}"; then
        return 1
    fi

    if ! execute_command "update project $PROJECT_NAME" ./main/main.go Sls UpdateProject --project "$PROJECT_NAME" --body "{\"description\":\"this is test for update\"}"; then
        return 1
    fi

    if ! execute_command "get project $PROJECT_NAME" ./main/main.go Sls GetProject --project "$PROJECT_NAME"; then
        return 1
    fi

    if ! execute_command "list project" ./main/main.go Sls ListProject; then
        return 1
    fi

    if ! execute_command "delete project $PROJECT_NAME" ./main/main.go Sls DeleteProject --project "$PROJECT_NAME"; then
        return 1
    fi
}

sls_metric_store_test() {
    TIMESTAMP=$(date +"%Y%m%d-%H%M")
    PROJECT_NAME="mx-pro-$TIMESTAMP-$RANDOM"
    METRIC_STORE_NAME="mx-ms-$TIMESTAMP-$RANDOM"
    DESCRIPTION="this is test"

    echo "###### Try to test metric store $METRIC_STORE_NAME for project $PROJECT_NAME ######"
	if ! execute_command "create project $PROJECT_NAME" ./main/main.go Sls CreateProject --body "{\"description\":\"$DESCRIPTION\",\"projectName\":\"$PROJECT_NAME\"}"; then
        return 1
    fi

    if ! execute_command "create metric store $METRIC_STORE_NAME" ./main/main.go Sls CreateMetricStore --project "$PROJECT_NAME" --body "{\"name\":\"$METRIC_STORE_NAME\",\"ttl\":7,\"shardCount\":2}"; then
        return 1
    fi

    if ! execute_command "get metric store $METRIC_STORE_NAME" ./main/main.go Sls GetMetricStore --project "$PROJECT_NAME" --name "$METRIC_STORE_NAME"; then
        return 1
    fi

    if ! execute_command "update metric store $METRIC_STORE_NAME" ./main/main.go Sls UpdateMetricStore --project "$PROJECT_NAME" --name "$METRIC_STORE_NAME" --body "{\"ttl\":3}"; then
        return 1
    fi

    if ! execute_command "list metric store" ./main/main.go Sls ListMetricStores --project "$PROJECT_NAME"; then
        return 1
    fi

    if ! execute_command "get metric store $METRIC_STORE_NAME" ./main/main.go Sls GetMetricStore --project "$PROJECT_NAME" --name "$METRIC_STORE_NAME"; then
        return 1
    fi

    if ! execute_command "delete metric store $METRIC_STORE_NAME" ./main/main.go Sls DeleteMetricStore --project "$PROJECT_NAME" --name "$METRIC_STORE_NAME"; then
        return 1
    fi

    if ! execute_command "delete project $PROJECT_NAME" ./main/main.go Sls DeleteProject --project "$PROJECT_NAME"; then
        return 1
    fi
}

echo "###### Start to test sls project ######"
sls_project_test
echo "###### End to test sls project ######"

echo "###### Start to test sls metric store ######"
sls_metric_store_test
echo "###### End to test sls metric store ######"
exit