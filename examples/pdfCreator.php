<?
$resultDir = "/Users/fans/go/src/bitbucket.org/sotavant/rabbitOxpa/examples/result/tasks";

$n = rand(5, 10);
sleep($n);
$data = json_decode($argv[1]);
$fileName = $data->fileName;
$taskId = $data->taskId;
$taskDir = $resultDir . "/" . $taskId;
try {
    if (!is_dir($taskDir))
        mkdir($taskDir);

    file_put_contents($taskDir . '/' . $fileName, 'test');

} catch (Exception $e) {
    echo $e->getMessage();
}
