const proxyTarget = 'http://127.0.0.1:5032'
const wsTarget = proxyTarget.replace('http', 'ws')

module.exports = {
    publicPath: './',
    outputDir: 'dist',

    // 放置静态资源的地方 (js/css/img/font/...)
    assetsDir: 'static',

    css: {
        loaderOptions: {
            sass: {
                sassOptions: {
                    silenceDeprecations: ['legacy-js-api']
                }
            },
            scss: {
                sassOptions: {
                    silenceDeprecations: ['legacy-js-api']
                }
            }
        }
    },

    configureWebpack: {
        performance: {
            hints: false
        }
    },

    lintOnSave: true,

    // 是否为生产环境构建生成 source map？
    productionSourceMap: false,

    // 在多核机器下会默认开启。
    parallel: require('os').cpus().length > 1,

    // 配置 webpack-dev-server 行为。
    devServer: {
        host: '0.0.0.0',
        port: 8257,
        https: false,
        open: process.platform === 'darwin',
        // 查阅 https://github.com/vuejs/vue-docs-zh-cn/blob/master/vue-cli/cli-service.md#配置代理
        proxy: {
            '/api': {
                target: proxyTarget,
                changeOrigin: true,
                ws: true,
                pathRewrite: (path) => {
                    if (/^\/api\/(status|login)(\/|$)/.test(path)) {
                        return path
                    }
                    return path.replace(/^\/api/, '')
                }
            },
            '/ws': {
                target: wsTarget,
                changeOrigin: true,
                ws: true,
                pathRewrite: {
                    '^/ws': ''
                }
            }
        }
    }
}
