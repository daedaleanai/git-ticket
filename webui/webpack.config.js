const path = require('path');
const autoprefixer = require('autoprefixer')

module.exports = {
    mode: "production",
    entry: {
        shared: './static/src/js/shared.js',
        home: './static/src/js/home.js',
        checklist: './static/src/js/checklist.js',
        create: './static/src/js/create.ts',
        ticket: './static/src/js/ticket.js',
    },
    output: {
        path: path.resolve(__dirname, 'static/dist'),
        filename: '[name].js',
    },
    module: {
        noParse: /\/node_modules\/process\//,
        rules: [
            {
                test: /\.tsx?$/,
                use: 'ts-loader',
                exclude: /node_modules/,
            },
            {
                test: /\.css$/,
                use: [ 'style-loader', 'css-loader']
            },
            {
                test: /\.(scss)$/,
                use: [
                    {loader: 'style-loader'},
                    {loader: 'css-loader'},
                    {
                        loader: 'postcss-loader',
                        options: {
                            postcssOptions: {
                                plugins: [
                                    autoprefixer
                                ]
                            }
                        }
                    },
                    {loader: 'sass-loader'}
                ]
            },
        ],
    },
    resolve: {
        extensions: ['.tsx', '.ts', '.js'],
    },
};