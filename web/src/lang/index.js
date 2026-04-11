import Vue from 'vue'
import VueI18n from 'vue-i18n'
import locale from 'element-ui/lib/locale'
import elementEnLocale from 'element-ui/lib/locale/lang/en'
import elementZhLocale from 'element-ui/lib/locale/lang/zh-CN'
import enLocale from './en'
import zhLocale from './zh'

Vue.use(VueI18n)

const messages = {
    en: {
        ...enLocale,
        ...elementEnLocale
    },
    zh: {
        ...zhLocale,
        ...elementZhLocale
    }
}
export function getLanguage() {
    const chooseLanguage = localStorage.getItem('language')
    if (chooseLanguage && Object.prototype.hasOwnProperty.call(messages, chooseLanguage)) {
        return chooseLanguage
    }
    return 'zh'
}
const i18n = new VueI18n({
    locale: getLanguage(),
    messages,
    silentTranslationWarn: true
})

locale.i18n((key, value) => i18n.t(key, value))

export default i18n
