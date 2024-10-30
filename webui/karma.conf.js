// Karma configuration file, see link for more information
// https://karma-runner.github.io/1.0/config/configuration-file.html

module.exports = function (config) {
    config.set({
        basePath: '',
        frameworks: ['jasmine', '@angular-devkit/build-angular'],
        plugins: [
            require('karma-chrome-launcher'),
            require('karma-coverage-istanbul-reporter'),
            require('karma-jasmine'),
            require('karma-jasmine-html-reporter'),
            require('karma-junit-reporter'),
            require('karma-spec-reporter'),
            require('@angular-devkit/build-angular/plugins/karma'),
        ],
        client: {
            clearContext: false, // leave Jasmine Spec Runner output visible in browser
        },
        coverageIstanbulReporter: {
            dir: require('path').join(__dirname, './coverage/stork'),
            reports: ['html', 'lcovonly', 'text-summary'],
            fixWebpackSourcePaths: true,
        },
        junitReporter: {
            outputFile: 'junit.xml',
            useBrowserName: false,
        },
        reporters: ['junit', 'kjhtml', 'progress', 'spec'],
        port: 9876,
        colors: true,
        logLevel: config.LOG_INFO,
        autoWatch: true,
        browsers: ['Chrome'],
        customLaunchers: {
            ChromeNoSandboxHeadless: {
                base: 'ChromeHeadless',
                flags: ['--no-sandbox'],
            },
        },
        singleRun: false,
        restartOnFileChange: true,
    })
}
