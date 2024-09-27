const { exec } = require('child_process');

const interval = 60000; // 5 seconds
const command = 'go run node_exporter.go --log.level debug';

setInterval(() => {
    exec(command, (error, stdout, stderr) => {
        if (error) {
            console.error(`执行出错: ${error}`);
            return;
        }
        console.log(`输出: ${stdout}`);
        if (stderr) {
            console.error(`错误: ${stderr}`);
        }
    });
}, interval);