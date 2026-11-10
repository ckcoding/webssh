import Vue from 'vue'

import element from '@/utils/element'

import '@/styles/index.scss'

import App from '@/App'
import store from '@/store/index'

import i18n from './lang/index'

Vue.use(element)

Vue.config.productionTip = false

/* eslint-disable no-new */
new Vue({
    el: '#app',
    i18n,
    store,
    render: h => h(App)
})
