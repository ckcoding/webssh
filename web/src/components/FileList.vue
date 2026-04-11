<template>
    <div class="file-browser-wrapper">
        <el-button type="primary" size="small" icon="el-icon-folder-opened" :disabled="!hasActiveConnection || interactionLocked"
            @click="openFileBrowser">
            {{ $t('FileBrowser') }}
        </el-button>

        <el-dialog :title="$t('FileBrowser') + ' - ' + (this.$store.state.sshInfo.host || '')"
            :visible.sync="dialogVisible" top="0" :width="dialogWidth"
            custom-class="premium-file-dialog centered-dialog"
            :close-on-click-modal="false"
            :close-on-press-escape="false"
            :show-close="!interactionLocked"
            :before-close="handleMainDialogBeforeClose"
            append-to-body>
            <div class="file-browser-content" v-loading="interactionLocked" :element-loading-text="loadingText" element-loading-spinner="el-icon-loading">
                <div class="toolbar-container">
                    <el-input v-model="currentPath" :disabled="interactionLocked" @keyup.enter.native="getFileList()" class="path-input"
                        prefix-icon="el-icon-folder-opened"></el-input>

                    <div class="btns-wrapper">
                        <el-button-group>
                            <el-button type="primary" size="small" icon="el-icon-arrow-up"
                                :disabled="interactionLocked || currentPath === '/'" @click="upDirectory()"></el-button>
                            <el-button type="primary" size="small" icon="el-icon-refresh"
                                :disabled="interactionLocked" @click="getFileList({ force: true })"></el-button>
                        </el-button-group>

                        <el-dropdown @command="handleCreateCommand" trigger="click" :disabled="interactionLocked">
                            <el-button type="primary" size="small" icon="el-icon-circle-plus-outline" :disabled="interactionLocked">
                                {{ $t('Create') }}<i class="el-icon-arrow-down el-icon--right"></i>
                            </el-button>
                            <el-dropdown-menu slot="dropdown" class="premium-dropdown">
                                <el-dropdown-item command="file" icon="el-icon-document-add">{{ $t('NewFile') }}</el-dropdown-item>
                                <el-dropdown-item command="folder" icon="el-icon-folder-add">{{ $t('NewFolder') }}</el-dropdown-item>
                            </el-dropdown-menu>
                        </el-dropdown>

                        <el-dropdown @command="handleUploadCommand" trigger="click" :disabled="interactionLocked">
                            <el-button type="success" size="small" icon="el-icon-upload2" :disabled="interactionLocked">
                                {{ $t('Upload') }}<i class="el-icon-arrow-down el-icon--right"></i>
                            </el-button>
                            <el-dropdown-menu slot="dropdown" class="premium-dropdown">
                                <el-dropdown-item command="file" icon="el-icon-document">{{ $t('uploadFile')
                                }}</el-dropdown-item>
                                <el-dropdown-item command="folder" icon="el-icon-folder">{{ $t('uploadFolder')
                                }}</el-dropdown-item>
                            </el-dropdown-menu>
                        </el-dropdown>
                    </div>
                </div>

                <div v-if="loading && fileList.length === 0" class="file-empty-state loading-state" :style="{ height: clientHeight + 'px' }">
                    <div class="folder-loader">
                        <i class="el-icon-folder folder-base"></i>
                        <i class="el-icon-loading folder-spinner"></i>
                    </div>
                    <div class="file-empty-text">{{ $t('loadingDirectory') || 'Processing...' }}</div>
                    <div class="file-empty-hint">{{ currentPath }}</div>
                </div>

                <el-table v-else-if="fileList.length > 0" :data="fileList" :height="clientHeight" @row-click="rowClick"
                    class="premium-table" :class="{ 'is-locked': interactionLocked }" style="width: 100%">
                    <el-table-column :label="$t('Name')" min-width="260" sortable :sort-method="nameSort">
                        <template slot-scope="scope">
                            <div v-if="scope.row.IsDir" class="file-icon-text folder">
                                <div class="file-icon-wrapper folder-icon">
                                    <i class="el-icon-folder-opened"></i>
                                </div>
                                <div class="file-name-stack">
                                    <span class="name-text">{{ scope.row.Name }}</span>
                                </div>
                            </div>
                            <div v-else class="file-icon-text file">
                                <div class="file-icon-wrapper" :class="fileTypeClass(scope.row.Name)">
                                    <i class="el-icon-document"></i>
                                </div>
                                <div class="file-name-stack">
                                    <span class="name-text">{{ scope.row.Name }}</span>
                                    <div class="file-capability-row">
                                        <span v-for="tag in fileCapabilityTags(scope.row)" :key="`${scope.row.Name}-${tag.key}`"
                                            class="capability-tag" :class="`capability-${tag.type}`">
                                            {{ tag.label }}
                                        </span>
                                    </div>
                                </div>
                            </div>
                        </template>
                    </el-table-column>
                    <el-table-column :label="$t('Size')" width="140" align="center">
                        <template slot-scope="scope">
                            <span class="size-text">{{ scope.row.Size }}</span>
                        </template>
                    </el-table-column>
                    <el-table-column :label="$t('ModifiedTime')" width="200" align="center" sortable>
                        <template slot-scope="scope">
                            <span class="time-text">{{ scope.row.ModifyTime }}</span>
                        </template>
                    </el-table-column>
                    <el-table-column :label="$t('Actions')" width="170" align="center">
                        <template slot-scope="scope">
                            <template v-if="scope.row.IsDir">
                                <el-button type="text" size="mini" icon="el-icon-right" :disabled="interactionLocked"
                                    @click.stop="rowClick(scope.row)" class="action-btn open-action"
                                    :title="$t('Open')"></el-button>
                            </template>
                            <template v-else>
                                <el-button v-if="isEditableFile(scope.row)" type="text" size="mini" icon="el-icon-edit"
                                    :disabled="interactionLocked" @click.stop="openEditor(scope.row)"
                                    class="action-btn edit-action" :title="$t('Edit')"></el-button>
                                <el-button v-if="isPreviewableFile(scope.row) || isPlayableFile(scope.row)" type="text" size="mini"
                                    :icon="previewActionIcon(scope.row)" :disabled="interactionLocked"
                                    @click.stop="openPreview(scope.row)" class="action-btn preview-action"
                                    :title="previewActionTitle(scope.row)"></el-button>
                                <el-button v-if="!isEditableFile(scope.row) && !isPreviewableFile(scope.row) && !isPlayableFile(scope.row)"
                                    type="text" size="mini" icon="el-icon-lock" disabled class="action-btn readonly-action"
                                    :title="$t('downloadOnlyFile')"></el-button>
                                <el-button type="text" size="mini" icon="el-icon-download"
                                    :disabled="interactionLocked" @click.stop="downloadFileDirect(scope.row)"
                                    :title="$t('Download')" class="action-btn download-action"></el-button>
                            </template>
                        </template>
                    </el-table-column>
                </el-table>

                <div v-else class="file-empty-state" :style="{ height: clientHeight + 'px' }">
                    <i class="el-icon-folder-opened"></i>
                    <div class="file-empty-text">{{ $t('emptyDirectory') }}</div>
                    <div class="file-empty-hint">{{ currentPath }}</div>
                </div>
            </div>

            <el-dialog custom-class="premium-editor-dialog centered-dialog" :title="$t('editFileTitle') + ': ' + editingFileName"
                :visible.sync="editorVisible"
                :close-on-click-modal="false"
                :close-on-press-escape="false"
                :show-close="!(editorLoading || saveLoading || monacoLoading)"
                :before-close="handleEditorBeforeClose"
                @closed="handleEditorClosed"
                append-to-body width="80%" top="0">
                <div v-loading="editorLoading || monacoLoading" :element-loading-text="$t('openingFile')" element-loading-spinner="el-icon-loading">
                    <div ref="monacoContainer" style="width: 100%; height: 65vh; min-height: 500px; border-bottom: 1px solid #e2e8f0;"></div>
                </div>
                <div slot="footer" class="dialog-footer">
                    <el-button size="small" :disabled="editorLoading || saveLoading || monacoLoading" @click="editorVisible = false">{{ $t('Cancel') }}</el-button>
                    <el-button size="small" type="primary" :loading="saveLoading" :disabled="editorLoading || monacoLoading" @click="saveContent">{{ $t('saveChanges') }}</el-button>
                </div>
            </el-dialog>

            <el-dialog custom-class="premium-preview-dialog centered-dialog" :title="$t('previewFileTitle') + ': ' + previewFileName"
                :visible.sync="previewVisible"
                :close-on-click-modal="false"
                :close-on-press-escape="false"
                :show-close="!previewLoading"
                :before-close="handlePreviewBeforeClose"
                append-to-body width="78%" top="0">
                <div class="preview-stage" v-loading="previewLoading" :element-loading-text="$t('loadingPreview')" element-loading-spinner="el-icon-loading">
                    <img v-if="previewKind === 'image'" :src="previewUrl" class="preview-image" @load="handlePreviewReady" @error="handlePreviewError" />
                    <video v-else-if="previewKind === 'video'" :src="previewUrl" class="preview-video" controls autoplay playsinline
                        @loadeddata="handlePreviewReady" @error="handlePreviewError"></video>
                    <audio v-else-if="previewKind === 'audio'" :src="previewUrl" class="preview-audio" controls autoplay
                        @loadeddata="handlePreviewReady" @error="handlePreviewError"></audio>
                    <iframe v-else-if="previewKind === 'pdf'" :src="previewUrl" class="preview-frame" @load="handlePreviewReady"></iframe>
                </div>
                <div slot="footer" class="dialog-footer">
                    <el-button size="small" :disabled="previewLoading" @click="resetPreviewState">{{ $t('Cancel') }}</el-button>
                    <el-button size="small" type="primary" :disabled="previewLoading" @click="downloadCurrentPreview">{{ $t('Download') }}</el-button>
                </div>
            </el-dialog>

            <el-dialog custom-class="uploadContainer centered-dialog" :title="$t(this.titleTip)" :visible.sync="uploadVisible"
                :close-on-click-modal="false"
                :close-on-press-escape="false"
                :show-close="activeUploads === 0"
                :before-close="handleUploadBeforeClose"
                append-to-body :width="uploadWidth">
                <el-upload ref="upload" multiple drag :action="uploadUrl" :data="uploadData"
                    :before-upload="beforeUpload" :on-progress="uploadProgress" :on-success="uploadSuccess" :on-error="uploadError"
                    :disabled="interactionLocked">
                    <i class="el-icon-upload"></i>
                    <div class="el-upload__text">{{ $t(this.selectTip) }}</div>
                    <div class="el-upload__tip" slot="tip">{{ this.uploadTip }}</div>
                </el-upload>
            </el-dialog>
        </el-dialog>
    </div>
</template>

<script>
import { MessageBox } from 'element-ui'
import { fileList, getFileContent, saveFileContent, createFile as createRemoteFile, createDirectory as createRemoteDirectory } from '@/api/file'
import { mapState } from 'vuex'

const DIRECTORY_CACHE_TTL = 10 * 1000
const FILE_CONTENT_CACHE_TTL = 30 * 1000
const directoryCache = new Map()
const fileContentCache = new Map()
const EDITABLE_EXTS = new Set(['js', 'ts', 'jsx', 'tsx', 'vue', 'py', 'go', 'rs', 'java', 'rb', 'c', 'cc', 'cpp', 'h', 'hpp', 'php', 'sh', 'bash', 'zsh', 'css', 'scss', 'less', 'html', 'htm', 'lua', 'swift', 'kt', 'json', 'yaml', 'yml', 'toml', 'xml', 'ini', 'conf', 'cfg', 'env', 'properties', 'sql', 'csv', 'md', 'txt', 'rst', 'log', 'out', 'err'])
const EDITABLE_FILENAMES = new Set(['dockerfile', 'makefile', '.env', '.gitignore', '.gitattributes', '.editorconfig', '.npmrc', '.yarnrc', '.zshrc', '.bashrc', '.profile', 'readme', 'readme.md'])
const IMAGE_EXTS = new Set(['png', 'jpg', 'jpeg', 'gif', 'svg', 'webp', 'ico', 'bmp', 'avif'])
const VIDEO_EXTS = new Set(['mp4', 'webm', 'mov', 'm4v', 'mkv', 'ogv'])
const AUDIO_EXTS = new Set(['mp3', 'wav', 'ogg', 'm4a', 'aac', 'flac'])
const PREVIEW_EXTS = new Set(['pdf'])

function normalizePath(path) {
    let nextPath = String(path || '').trim()
    if (nextPath === '') {
        return '/'
    }
    nextPath = nextPath.replace(/\/+/g, '/')
    if (nextPath.length > 1) {
        nextPath = nextPath.replace(/\/+$/, '')
    }
    return nextPath || '/'
}

function buildCacheKey(sshInfo, path) {
    return `${sshInfo}::${path}`
}

function cloneFileEntries(list) {
    return Array.isArray(list) ? list.map(item => ({ ...item })) : []
}

function getCachedValue(cache, key, ttl) {
    const entry = cache.get(key)
    if (!entry) {
        return null
    }
    if (Date.now() - entry.timestamp > ttl) {
        cache.delete(key)
        return null
    }
    return entry.value
}

function setCachedValue(cache, key, value) {
    cache.set(key, {
        timestamp: Date.now(),
        value
    })
}

function getFileExtension(name) {
    const normalized = String(name || '').trim().toLowerCase()
    const lastDot = normalized.lastIndexOf('.')
    if (lastDot <= 0 || lastDot === normalized.length - 1) {
        return ''
    }
    return normalized.slice(lastDot + 1)
}

function getFileCapabilities(name) {
    const normalized = String(name || '').trim().toLowerCase()
    const ext = getFileExtension(normalized)
    const editable = EDITABLE_EXTS.has(ext) || EDITABLE_FILENAMES.has(normalized)
    if (IMAGE_EXTS.has(ext)) {
        return { editable: false, previewable: true, playable: false, previewKind: 'image' }
    }
    if (VIDEO_EXTS.has(ext)) {
        return { editable: false, previewable: false, playable: true, previewKind: 'video' }
    }
    if (AUDIO_EXTS.has(ext)) {
        return { editable: false, previewable: false, playable: true, previewKind: 'audio' }
    }
    if (PREVIEW_EXTS.has(ext)) {
        return { editable: false, previewable: true, playable: false, previewKind: 'pdf' }
    }
    if (editable) {
        return { editable: true, previewable: false, playable: false, previewKind: '' }
    }
    return { editable: false, previewable: false, playable: false, previewKind: '' }
}

export default {
    name: 'FileList',
    data() {
        return {
            uploadVisible: false,
            dialogVisible: false,
            fileList: [],
            downloadFilePath: '',
            currentPath: '',
            pathSpan: 18,
            clientHeight: 0,
            selectTip: 'clickSelectFile',
            titleTip: 'uploadFile',
            uploadTip: '',
            dialogWidth: '50%',
            uploadWidth: '32%',
            nameWidth: 260,
            progressPercent: 0,
            uploadData: {
                sshInfo: '',
                path: '',
                id: '',
                dir: ''
            },
            resizeHandler: null,
            loading: false,
            editorVisible: false,
            editorLoading: false,
            monacoLoading: false,
            saveLoading: false,
            previewVisible: false,
            previewLoading: false,
            previewUrl: '',
            previewKind: '',
            previewFileName: '',
            previewFilePath: '',
            editingContent: '',
            editingFileName: '',
            editingFilePath: '',
            createLoading: false,
            createTargetType: '',
            lineCount: 1,
            activeUploads: 0,
            listRequestToken: 0,
            editorRequestToken: 0,
            uploadRefreshTimer: null
        }
    },
    created() {
        this.resizeDialog()
    },
    mounted() {
        this.resizeDialog()
        this.resizeHandler = () => {
            this.resizeDialog()
        }
        window.addEventListener('resize', this.resizeHandler)
    },
    computed: {
        ...mapState(['currentTab']),
        hasActiveConnection() {
            return Boolean(this.currentTab && this.currentTab.name)
        },
        interactionLocked() {
            return this.loading || this.editorLoading || this.saveLoading || this.previewLoading || this.createLoading || this.activeUploads > 0
        },
        loadingText() {
            if (this.activeUploads > 0) {
                return this.$t('uploadingFiles')
            }
            if (this.createLoading) {
                return this.$t(this.createTargetType === 'folder' ? 'creatingFolder' : 'creatingFile')
            }
            if (this.saveLoading) {
                return this.$t('savingFile')
            }
            if (this.editorLoading) {
                return this.$t('openingFile')
            }
            if (this.previewLoading) {
                return this.$t('loadingPreview')
            }
            if (this.loading) {
                return this.$t('loadingDirectory')
            }
            return this.$t('loadingDirectory')
        },
        uploadUrl() {
            const prefix = process.env.NODE_ENV === 'production' ? `${location.origin}` : 'api'
            const token = localStorage.getItem('webssh_token')
            if (!token) {
                return `${prefix}/file/upload`
            }
            return `${prefix}/file/upload?${new URLSearchParams({ token }).toString()}`
        }
    },
    watch: {
        currentTab: function () {
            this.fileList = []
            this.currentPath = normalizePath(this.currentTab && this.currentTab.path)
            const cached = this.getDirectoryCache(this.currentPath)
            if (cached !== null) {
                this.fileList = cached
            }
            if (this.dialogVisible && !this.interactionLocked && this.hasActiveConnection) {
                this.getFileList()
            }
        }
    },
    methods: {
        openFileBrowser() {
            if (!this.hasActiveConnection || this.interactionLocked) {
                return
            }
            this.currentPath = normalizePath(this.currentTab && this.currentTab.path)
            this.dialogVisible = true
            this.getFileList({ clearBeforeLoad: true })
        },
        handleMainDialogBeforeClose(done) {
            if (this.interactionLocked) {
                this.$message.warning(this.$t('fileOperationLocked'))
                return
            }
            done()
        },
        handleEditorBeforeClose(done) {
            if (this.editorLoading || this.saveLoading || this.monacoLoading) {
                this.$message.warning(this.$t('fileOperationLocked'))
                return
            }
            done()
        },
        handleEditorClosed() {
            if (this.monacoEditor) {
                this.monacoEditor.dispose()
                this.monacoEditor = null
            }
        },
        loadMonaco() {
            return new Promise((resolve) => {
                if (window.monaco) {
                    resolve(window.monaco)
                    return
                }
                const script = document.createElement('script')
                script.src = 'https://cdnjs.cloudflare.com/ajax/libs/monaco-editor/0.36.1/min/vs/loader.min.js'
                script.onload = () => {
                    window.require.config({ paths: { 'vs': 'https://cdnjs.cloudflare.com/ajax/libs/monaco-editor/0.36.1/min/vs' } })
                    window.require(['vs/editor/editor.main'], () => {
                        resolve(window.monaco)
                    })
                }
                document.head.appendChild(script)
            })
        },
        initMonaco(filename) {
            if (this.monacoEditor) {
                this.monacoEditor.dispose()
                this.monacoEditor = null
            }
            const container = this.$refs.monacoContainer
            if (!container) return

            const extParts = filename.split('.')
            const extStr = extParts.length > 1 ? extParts.pop().toLowerCase() : ''
            let language = 'plaintext'
            const langMap = {
                js: 'javascript', ts: 'typescript', vue: 'html', html: 'html', css: 'css', scss: 'scss', less: 'less',
                go: 'go', py: 'python', sh: 'shell', bash: 'shell', zsh: 'shell', json: 'json',
                yaml: 'yaml', yml: 'yaml', xml: 'xml', sql: 'sql', md: 'markdown', cpp: 'cpp', c: 'c',
                java: 'java', php: 'php', rb: 'ruby', rs: 'rust', ini: 'ini', conf: 'ini'
            }
            if (langMap[extStr]) {
                language = langMap[extStr]
            } else if (filename.toLowerCase() === 'dockerfile') {
                language = 'dockerfile'
            }

            this.monacoEditor = window.monaco.editor.create(container, {
                value: this.editingContent,
                language: language,
                theme: 'vs',
                automaticLayout: true,
                fontSize: 14,
                fontFamily: "'JetBrains Mono', 'Fira Code', 'Courier New', monospace",
                minimap: { enabled: false },
                scrollBeyondLastLine: false,
                renderLineHighlight: 'all',
                padding: { top: 16 }
            })

            this.monacoEditor.onDidChangeModelContent(() => {
                this.editingContent = this.monacoEditor.getValue()
            })
        },
        handlePreviewBeforeClose(done) {
            if (this.previewLoading) {
                this.$message.warning(this.$t('fileOperationLocked'))
                return
            }
            this.resetPreviewState()
            done()
        },
        handleUploadBeforeClose(done) {
            if (this.activeUploads > 0) {
                this.$message.warning(this.$t('fileOperationLocked'))
                return
            }
            done()
        },
        getCapabilities(row) {
            return getFileCapabilities(row && row.Name)
        },
        isEditableFile(row) {
            return this.getCapabilities(row).editable
        },
        isPreviewableFile(row) {
            return this.getCapabilities(row).previewable
        },
        isPlayableFile(row) {
            return this.getCapabilities(row).playable
        },
        fileCapabilityTags(row) {
            const caps = this.getCapabilities(row)
            const tags = []
            if (caps.editable) {
                tags.push({ key: 'editable', label: this.$t('editableFile'), type: 'editable' })
            }
            if (caps.previewable) {
                tags.push({ key: 'preview', label: this.$t('previewableFile'), type: 'preview' })
            }
            if (caps.playable) {
                tags.push({ key: 'play', label: this.$t('playableFile'), type: 'play' })
            }
            if (tags.length === 0) {
                tags.push({ key: 'download', label: this.$t('downloadOnlyFile'), type: 'download' })
            }
            return tags
        },
        previewActionIcon(row) {
            return this.isPlayableFile(row) ? 'el-icon-video-play' : 'el-icon-view'
        },
        previewActionTitle(row) {
            return this.isPlayableFile(row) ? this.$t('Play') : this.$t('Preview')
        },
        buildFilePath(row) {
            const base = this.currentPath.replace(/\/+$/, '')
            return normalizePath(`${base}/${row.Name}`)
        },
        buildFileUrl(path, inline = false) {
            const prefix = process.env.NODE_ENV === 'production' ? `${location.origin}` : 'api'
            const params = {
                path,
                sshInfo: this.$store.getters.sshReq
            }
            if (inline) {
                params.inline = '1'
            }
            const token = localStorage.getItem('webssh_token')
            if (token) {
                params.token = token
            }
            return `${prefix}/file/download?${new URLSearchParams(params).toString()}`
        },
        resetPreviewState() {
            this.previewVisible = false
            this.previewLoading = false
            this.previewUrl = ''
            this.previewKind = ''
            this.previewFileName = ''
            this.previewFilePath = ''
        },
        handlePreviewReady() {
            this.previewLoading = false
        },
        handlePreviewError() {
            this.previewLoading = false
            this.$message.error(this.$t('previewFailed'))
            this.resetPreviewState()
        },
        downloadCurrentPreview() {
            if (this.previewFilePath === '') {
                return
            }
            this.downloadFilePath = this.previewFilePath
            this.downloadFile()
        },
        getDirectoryCache(path) {
            const sshInfo = this.$store.getters.sshReq
            const cacheKey = buildCacheKey(sshInfo, normalizePath(path))
            const cached = getCachedValue(directoryCache, cacheKey, DIRECTORY_CACHE_TTL)
            return cached === null ? null : cloneFileEntries(cached)
        },
        setDirectoryCache(path, list) {
            const sshInfo = this.$store.getters.sshReq
            const cacheKey = buildCacheKey(sshInfo, normalizePath(path))
            setCachedValue(directoryCache, cacheKey, cloneFileEntries(list))
        },
        invalidateDirectoryCache(path) {
            const sshInfo = this.$store.getters.sshReq
            directoryCache.delete(buildCacheKey(sshInfo, normalizePath(path)))
        },
        getFileContentCache(path) {
            const sshInfo = this.$store.getters.sshReq
            return getCachedValue(fileContentCache, buildCacheKey(sshInfo, normalizePath(path)), FILE_CONTENT_CACHE_TTL)
        },
        setFileContentCache(path, content) {
            const sshInfo = this.$store.getters.sshReq
            setCachedValue(fileContentCache, buildCacheKey(sshInfo, normalizePath(path)), content)
        },
        invalidateFileContentCache(path) {
            const sshInfo = this.$store.getters.sshReq
            fileContentCache.delete(buildCacheKey(sshInfo, normalizePath(path)))
        },
        scheduleDirectoryRefresh(path = this.currentPath) {
            const targetPath = normalizePath(path)
            if (this.uploadRefreshTimer !== null) {
                clearTimeout(this.uploadRefreshTimer)
            }
            this.uploadRefreshTimer = setTimeout(() => {
                this.uploadRefreshTimer = null
                this.invalidateDirectoryCache(targetPath)
                if (this.dialogVisible && normalizePath(this.currentPath) === targetPath) {
                    this.getFileList({ force: true })
                }
            }, 300)
        },
        resizeDialog() {
            const clientWidth = document.body.clientWidth
            const clientHeight = document.body.clientHeight
            const maxTableH = 650
            this.clientHeight = Math.min(clientHeight - 280, maxTableH)

            if (clientWidth < 600) {
                this.dialogWidth = '98%'
                this.uploadWidth = '100%'
            } else if (clientWidth < 1200) {
                this.dialogWidth = '85%'
                this.uploadWidth = '50%'
            } else {
                this.dialogWidth = '75%'
                this.uploadWidth = '30%'
            }
        },
        validateCreateName(value) {
            const name = String(value || '').trim()
            return name !== '' && name !== '.' && name !== '..' && !/[\\/]/.test(name)
        },
        async handleCreateCommand(type) {
            if (this.interactionLocked) {
                return
            }
            const isFolder = type === 'folder'
            const result = await MessageBox.prompt('', this.$t(isFolder ? 'createFolderTitle' : 'createFileTitle'), {
                closeOnClickModal: false,
                closeOnPressEscape: false,
                inputValidator: value => this.validateCreateName(value),
                inputErrorMessage: this.$t(isFolder ? 'invalidFolderName' : 'invalidFileName')
            }).catch(() => null)
            if (!result || typeof result.value !== 'string') {
                return
            }
            await this.createEntry(type, result.value.trim())
        },
        async createEntry(type, name) {
            if (this.createLoading) {
                return
            }
            this.createLoading = true
            this.createTargetType = type
            let createdRow = null
            try {
                const action = type === 'folder' ? createRemoteDirectory : createRemoteFile
                const res = await action({
                    path: this.currentPath,
                    name,
                    sshInfo: this.$store.getters.sshReq
                })
                if (res.Msg !== 'success') {
                    this.$message.error(`${this.$t(type === 'folder' ? 'createFolderFailed' : 'createFileFailed')}: ${res.Msg}`)
                    return
                }
                this.invalidateDirectoryCache(this.currentPath)
                await this.getFileList({ force: true })
                this.$message.success(this.$t(type === 'folder' ? 'createFolderSuccess' : 'createFileSuccess'))
                if (type === 'file' && this.isEditableFile({ Name: name })) {
                    createdRow = this.fileList.find(item => !item.IsDir && item.Name === name) || null
                }
            } catch (e) {
                this.$message.error(this.$t('networkRequestFailed'))
            } finally {
                this.createLoading = false
                this.createTargetType = ''
            }
            if (createdRow !== null) {
                this.openEditor(createdRow)
            }
        },
        openUploadDialog() {
            this.uploadTip = `${this.$t('uploadPath')}: ${this.currentPath}`
            this.uploadData = {
                sshInfo: this.$store.getters.sshReq,
                path: this.currentPath,
                id: '',
                dir: ''
            }
            this.uploadVisible = true
        },
        handleUploadCommand(cmd) {
            if (this.interactionLocked) {
                return
            }
            if (cmd === 'folder') {
                this.selectTip = 'clickSelectFolder'
                this.titleTip = 'uploadFolder'
            } else {
                this.selectTip = 'clickSelectFile'
                this.titleTip = 'uploadFile'
            }
            this.openUploadDialog()
            const isFolder = cmd === 'folder'
            const supported = this.webkitdirectorySupported()
            if (!supported) {
                if (isFolder) {
                    this.$message.warning(this.$t('browserNotSupport'))
                }
                return
            }
            this.$nextTick(() => {
                const input = document.getElementsByClassName('el-upload__input')[0]
                if (input) {
                    input.webkitdirectory = isFolder
                }
            })
        },
        webkitdirectorySupported() {
            return 'webkitdirectory' in document.createElement('input')
        },
        beforeUpload(file) {
            this.activeUploads += 1
            this.uploadTip = `${this.$t('uploading')} ${file.name} ${this.$t('to')} ${this.currentPath}, ${this.$t('notCloseWindows')}..`
            this.uploadData.sshInfo = this.$store.getters.sshReq
            this.uploadData.path = this.currentPath
            this.uploadData.id = file.uid
            const dirPath = file.webkitRelativePath
            this.uploadData.dir = dirPath ? dirPath.substring(0, dirPath.lastIndexOf('/')) : ''
            return true
        },
        uploadSuccess(r, file) {
            this.uploadTip = `${file.name}${this.$t('uploadFinish')}!`
            this.activeUploads = Math.max(0, this.activeUploads - 1)
            this.scheduleDirectoryRefresh(this.currentPath)
        },
        uploadError() {
            this.activeUploads = Math.max(0, this.activeUploads - 1)
            this.$message.error(this.$t('uploadFailed'))
        },
        uploadProgress(e, f) {
            e.percent = e.percent / 2
            f.percentage = f.percentage / 2
            if (e.percent === 50) {
                const params = {
                    id: f.uid
                }
                const token = localStorage.getItem('webssh_token')
                if (token) {
                    params.token = token
                }
                const query = new URLSearchParams(params).toString()
                const ws = new WebSocket(`${(location.protocol === 'http:' ? 'ws' : 'wss')}://${location.host}${process.env.NODE_ENV === 'production' ? '' : '/ws'}/file/progress?${query}`)
                ws.onmessage = e1 => {
                    f.percentage = (f.size + Number(e1.data)) / (f.size * 2) * 100
                }
            }
        },
        nameSort(a, b) {
            return a.Name > b.Name
        },
        fileTypeClass(name) {
            if (!name) return 'file-generic'
            const ext = name.split('.').pop().toLowerCase()
            const CODE_EXTS = ['js', 'ts', 'jsx', 'tsx', 'vue', 'py', 'go', 'rs', 'java', 'rb', 'c', 'cpp', 'h', 'php', 'sh', 'bash', 'zsh', 'css', 'scss', 'less', 'html', 'htm', 'lua', 'swift', 'kt']
            const CONFIG_EXTS = ['json', 'yaml', 'yml', 'toml', 'xml', 'ini', 'conf', 'cfg', 'env', 'properties', 'dockerfile', 'nginx', 'service']
            const ARCHIVE_EXTS = ['zip', 'tar', 'gz', 'bz2', 'xz', '7z', 'rar', 'tgz', 'deb', 'rpm']
            const IMAGE_EXTS = ['png', 'jpg', 'jpeg', 'gif', 'svg', 'webp', 'ico', 'bmp', 'avif']
            const LOG_EXTS = ['log', 'out', 'err']
            const DATA_EXTS = ['sql', 'db', 'sqlite', 'csv', 'md', 'txt', 'rst', 'pem', 'key', 'crt', 'cert']
            if (CODE_EXTS.includes(ext)) return 'file-code'
            if (CONFIG_EXTS.includes(ext)) return 'file-config'
            if (ARCHIVE_EXTS.includes(ext)) return 'file-archive'
            if (IMAGE_EXTS.includes(ext)) return 'file-image'
            if (LOG_EXTS.includes(ext)) return 'file-log'
            if (DATA_EXTS.includes(ext)) return 'file-data'
            return 'file-generic'
        },
        rowClick(row) {
            if (this.interactionLocked) return

            let targetPath = ''
            if (row.IsDir) {
                // 统一收口路径，避免出现 /home/home 这类重复拼接。
                const base = this.currentPath.replace(/\/+$/, '')
                targetPath = `${base}/${row.Name}`.replace(/\/+/g, '/')
                if (targetPath === '') targetPath = '/'

                this.currentPath = targetPath
                this.getFileList({ clearBeforeLoad: true })
            } else {
                if (this.isEditableFile(row)) {
                    this.openEditor(row)
                    return
                }
                if (this.isPreviewableFile(row) || this.isPlayableFile(row)) {
                    this.openPreview(row)
                    return
                }
                this.$message.info(this.$t('downloadOnlyFileTip'))
            }
        },
        downloadFileDirect(row) {
            if (this.interactionLocked) {
                return
            }
            this.downloadFilePath = this.buildFilePath(row)
            this.downloadFile()
        },
        async getFileList(options = {}) {
            if (this.loading) return
            this.loading = true

            try {
                // 先标准化路径，再决定是否命中目录缓存。
                let path = normalizePath(this.currentPath)
                this.currentPath = path
                const force = Boolean(options.force)
                const requestToken = ++this.listRequestToken

                if (!force) {
                    const cachedList = this.getDirectoryCache(path)
                    if (cachedList !== null) {
                        this.fileList = cachedList
                        this.updatePath(this.currentPath)
                        return
                    }
                }
                
                if (options.clearBeforeLoad) {
                    this.fileList = []
                }

                const result = await fileList(this.currentPath, this.$store.getters.sshReq)
                if (requestToken !== this.listRequestToken) {
                    return
                }
                if (result.Msg === 'success') {
                    const nextList = result.Data.list === null ? [] : result.Data.list
                    this.fileList = nextList
                    this.setDirectoryCache(this.currentPath, nextList)
                    this.updatePath(this.currentPath)
                } else {
                    this.fileList = []
                    this.$message.error(result.Msg)
                }
            } finally {
                this.loading = false
            }
        },
        upDirectory() {
            if (this.interactionLocked) {
                return
            }
            if (this.currentPath === '/') {
                return
            }
            let pathList = this.currentPath.split('/')
            if (pathList[pathList.length - 1] === '') {
                pathList = pathList.slice(0, pathList.length - 2)
            } else {
                pathList = pathList.slice(0, pathList.length - 1)
            }
            this.currentPath = pathList.length === 1 ? '/' : pathList.join('/')
            this.getFileList({ clearBeforeLoad: true })
        },
        downloadFile() {
            const downloadUrl = this.buildFileUrl(this.downloadFilePath)
            window.open(downloadUrl)
        },
        updatePath(path) {
            const termList = this.$store.state.termList
            for (let i = 0; i < termList.length; ++i) {
                if (termList[i].name === this.currentTab.name) {
                    termList[i].path = path
                    break
                }
            }
            this.$store.commit('SET_TERMLIST', termList)
        },
        async openEditor(row) {
            if (this.interactionLocked || !this.isEditableFile(row)) {
                if (!this.isEditableFile(row)) {
                    this.$message.warning(this.$t('nonEditableFile'))
                }
                return
            }
            this.editingFileName = row.Name
            this.editingFilePath = this.buildFilePath(row)
            this.editorVisible = true
            this.editingContent = ''
            
            this.monacoLoading = true
            const requestToken = ++this.editorRequestToken
            this.editorLoading = true
            
            try {
                let content = ''
                const cachedContent = this.getFileContentCache(this.editingFilePath)
                if (cachedContent !== null) {
                    content = cachedContent
                } else {
                    const res = await getFileContent(this.editingFilePath, this.$store.getters.sshReq)
                    if (requestToken !== this.editorRequestToken) return
                    if (res.Msg === 'success') {
                        content = res.Data
                        this.setFileContentCache(this.editingFilePath, content)
                    } else {
                        this.$message.error(`${this.$t('openFileFailed')}: ${res.Msg}`)
                        this.editorVisible = false
                        return
                    }
                }
                
                this.editingContent = content
                
                await this.loadMonaco()
                
            } catch (e) {
                this.$message.error(this.$t('networkRequestFailed'))
                this.editorVisible = false
            } finally {
                this.editorLoading = false
                this.monacoLoading = false
                if (this.editorVisible) {
                    this.$nextTick(() => {
                        this.initMonaco(row.Name)
                    })
                }
            }
        },
        openPreview(row) {
            if (this.interactionLocked) {
                return
            }
            const capabilities = this.getCapabilities(row)
            if (!capabilities.previewable && !capabilities.playable) {
                this.$message.warning(this.$t('previewUnsupported'))
                return
            }
            this.previewFileName = row.Name
            this.previewFilePath = this.buildFilePath(row)
            this.previewKind = capabilities.previewKind
            this.previewUrl = this.buildFileUrl(this.previewFilePath, true)
            this.previewLoading = true
            this.previewVisible = true
        },
        async saveContent() {
            if (this.saveLoading) {
                return
            }
            this.saveLoading = true
            try {
                const res = await saveFileContent({
                    path: this.editingFilePath,
                    content: this.editingContent,
                    sshInfo: this.$store.getters.sshReq
                })
                if (res.Msg === 'success') {
                    this.setFileContentCache(this.editingFilePath, this.editingContent)
                    this.invalidateDirectoryCache(this.currentPath)
                    this.$message.success(this.$t('saveFileSuccess'))
                    this.editorVisible = false
                } else {
                    this.invalidateFileContentCache(this.editingFilePath)
                    this.$message.error(`${this.$t('saveFileFailed')}: ${res.Msg}`)
                }
            } catch (e) {
                this.invalidateFileContentCache(this.editingFilePath)
                this.$message.error(this.$t('networkSaveFailed'))
            } finally {
                this.saveLoading = false
            }
        }
    },
    beforeDestroy() {
        if (this.resizeHandler !== null) {
            window.removeEventListener('resize', this.resizeHandler)
            this.resizeHandler = null
        }
        if (this.uploadRefreshTimer !== null) {
            clearTimeout(this.uploadRefreshTimer)
            this.uploadRefreshTimer = null
        }
        this.resetPreviewState()
    }
}
</script>

<style scoped lang="scss">
.file-browser-wrapper {
    display: inline-block;
}

.file-browser-content {
    position: relative;
}

::v-deep .premium-file-dialog {
    border-radius: 16px;
    overflow: hidden;
    background: #ffffff;
    border: 1px solid #e2e8f0;
    box-shadow:
        0 25px 50px -12px rgba(0, 0, 0, 0.25),
        0 0 0 1px rgba(255, 255, 255, 0.5) inset;
    margin-top: 0 !important;

    &.centered-dialog {
        position: absolute;
        top: 50%;
        left: 50%;
        transform: translate(-50%, -50%);
        margin: 0 !important;
    }

    .el-dialog__header {
        padding: 16px 20px;
        background: linear-gradient(180deg, #f8fafc 0%, #f1f5f9 100%);
        border-bottom: 1px solid #e2e8f0;

        .el-dialog__title {
            font-weight: 700;
            color: #0f172a;
            font-size: 15px;
            letter-spacing: -0.01em;
        }
    }

    .el-dialog__body {
        padding: 16px 20px 20px;
    }

    .toolbar-container {
        display: flex;
        align-items: center;
        gap: 10px;
        width: 100%;
        margin-bottom: 14px;

        .path-input {
            flex: 1;

            .el-input__inner {
                background: #f8fafc;
                border: 1.5px solid #e2e8f0;
                border-radius: 10px;
                padding-left: 38px;
                padding-right: 12px;
                font-family: 'JetBrains Mono', 'Fira Code', 'SF Mono', monospace;
                font-size: 12.5px;
                height: 36px;
                line-height: 36px;
                color: #334155;

                &:focus {
                    background: #ffffff;
                    border-color: #3b82f6;
                    box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
                }
            }

            .el-input__prefix {
                left: 0;
                width: 38px;
                color: #64748b;
                display: flex;
                align-items: center;
                justify-content: center;
                height: 100%;

                i {
                    line-height: 1;
                    display: block;
                    font-size: 16px;
                }
            }
        }

        .btns-wrapper {
            display: flex;
            align-items: center;
            gap: 8px;
            flex-shrink: 0;

            .el-button-group {
                display: inline-flex;
            }
        }
    }

    .premium-table {
        border-radius: 12px;
        overflow: hidden;
        border: 1px solid #e2e8f0;
        background: #ffffff !important;

        &::before {
            display: none;
        }

        th {
            background: linear-gradient(180deg, #f8fafc 0%, #f1f5f9 100%) !important;
            padding: 10px 0;
            color: #475569;
            font-size: 12px;
            font-weight: 700;
            text-transform: uppercase;
            letter-spacing: 0.04em;
        }

        td {
            padding: 6px 0;
            border-bottom: 1px solid #f1f5f9 !important;
        }

        .file-icon-text {
            display: flex;
            align-items: center;
            gap: 10px;
            margin: 0;
            font-weight: 500;
            font-size: 13px;

            .file-icon-wrapper {
                width: 30px;
                height: 30px;
                border-radius: 8px;
                display: flex;
                align-items: center;
                justify-content: center;
                flex-shrink: 0;

                i {
                    font-size: 15px;
                }

                &.folder-icon {
                    background: rgba(59, 130, 246, 0.1);
                    i { color: #3b82f6; }
                }

                &.file-code {
                    background: rgba(168, 85, 247, 0.1);
                    i { color: #a855f7; }
                }

                &.file-config {
                    background: rgba(245, 158, 11, 0.1);
                    i { color: #f59e0b; }
                }

                &.file-archive {
                    background: rgba(239, 68, 68, 0.1);
                    i { color: #ef4444; }
                }

                &.file-image {
                    background: rgba(236, 72, 153, 0.1);
                    i { color: #ec4899; }
                }

                &.file-log {
                    background: rgba(107, 114, 128, 0.1);
                    i { color: #6b7280; }
                }

                &.file-data {
                    background: rgba(16, 185, 129, 0.1);
                    i { color: #10b981; }
                }

                &.file-generic {
                    background: rgba(148, 163, 184, 0.1);
                    i { color: #94a3b8; }
                }
            }

            &.folder {
                color: #1e40af;
                font-weight: 600;
            }

            &.file {
                color: #334155;
            }

            .name-text {
                white-space: nowrap;
                overflow: hidden;
                text-overflow: ellipsis;
            }
        }

        .file-name-stack {
            display: flex;
            flex-direction: column;
            gap: 4px;
            min-width: 0;
        }

        .file-capability-row {
            display: flex;
            flex-wrap: wrap;
            gap: 6px;
        }

        .capability-tag {
            display: inline-flex;
            align-items: center;
            padding: 2px 8px;
            border-radius: 999px;
            font-size: 10px;
            font-weight: 700;
            line-height: 1.4;
            letter-spacing: 0.04em;

            &.capability-editable {
                background: rgba(59, 130, 246, 0.12);
                color: #2563eb;
            }

            &.capability-preview {
                background: rgba(168, 85, 247, 0.12);
                color: #9333ea;
            }

            &.capability-play {
                background: rgba(245, 158, 11, 0.14);
                color: #d97706;
            }

            &.capability-download {
                background: rgba(148, 163, 184, 0.14);
                color: #64748b;
            }
        }

        .size-text {
            font-size: 12px;
            color: #64748b;
            font-variant-numeric: tabular-nums;
        }

        .time-text {
            font-size: 12px;
            color: #94a3b8;
            font-variant-numeric: tabular-nums;
        }

        .action-btn {
            padding: 4px 8px !important;
            font-size: 16px !important;
            border-radius: 6px !important;

            &.edit-action {
                color: #3b82f6 !important;
                &:hover {
                    background: rgba(59, 130, 246, 0.1) !important;
                    color: #2563eb !important;
                }
            }

            &.download-action {
                color: #10b981 !important;
                &:hover {
                    background: rgba(16, 185, 129, 0.1) !important;
                    color: #059669 !important;
                }
            }

            &.preview-action {
                color: #a855f7 !important;
                &:hover {
                    background: rgba(168, 85, 247, 0.1) !important;
                    color: #9333ea !important;
                }
            }

            &.open-action {
                color: #64748b !important;
                &:hover {
                    background: rgba(59, 130, 246, 0.1) !important;
                    color: #3b82f6 !important;
                }
            }

            &.readonly-action {
                color: #cbd5e1 !important;
            }
        }

        .el-table__row {
            cursor: pointer;

            &:hover > td {
                background-color: #f0f7ff !important;
            }
        }

        &.is-locked {
            .el-table__row {
                cursor: not-allowed;
            }

            .el-table__row:hover > td {
                background-color: #ffffff !important;
            }
        }
    }
}

.file-empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 60px 20px;
    color: #94a3b8;

    i {
        font-size: 48px;
        opacity: 0.4;
        margin-bottom: 12px;
    }

    .file-empty-text {
        font-size: 15px;
        font-weight: 600;
        color: #64748b;
        margin-bottom: 6px;
    }

    .file-empty-hint {
        font-size: 12px;
        color: #94a3b8;
        font-family: 'JetBrains Mono', 'Fira Code', monospace;
    }
}

.loading-state {
    .folder-loader {
        position: relative;
        width: 64px;
        height: 64px;
        margin-bottom: 16px;
        display: flex;
        align-items: center;
        justify-content: center;
        
        .folder-base {
            position: absolute;
            font-size: 64px;
            color: #e2e8f0;
            opacity: 0.6;
        }

        .folder-spinner {
            position: absolute;
            font-size: 24px;
            color: #3b82f6;
            z-index: 1;
        }
    }
}


::v-deep .uploadContainer {
    border-radius: 20px !important;
    overflow: hidden;
    box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.25) !important;
    margin: 0 !important;

    &.centered-dialog {
        position: absolute;
        top: 50%;
        left: 50%;
        transform: translate(-50%, -50%);
    }

    .el-dialog__header {
        background: linear-gradient(180deg, #f8fafc 0%, #f1f5f9 100%);
        border-bottom: 1px solid #e2e8f0;
        padding: 16px 20px;

        .el-dialog__title {
            font-weight: 700;
            color: #0f172a;
            font-size: 16px;
        }
    }

    .el-dialog__body {
        padding: 20px;
    }

    .el-upload {
        width: 100%;
    }

    .el-upload-dragger {
        width: 100%;
        border: 2px dashed #cbd5e1;
        background: #f8fafc;
        border-radius: 14px;
        height: 160px;
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;

        &:hover {
            border-color: #3b82f6;
            background: #f0f7ff;
        }

        .el-icon-upload {
            color: #94a3b8;
            font-size: 48px;
            margin: 0 0 10px;
            float: none;
            line-height: 1;
        }

        .el-upload__text {
            font-size: 14px;
            color: #64748b;
            line-height: 1.4;

            em {
                color: #3b82f6;
                font-style: normal;
                font-weight: 600;
            }
        }
    }

    .el-upload__tip {
        margin-top: 16px;
        padding: 12px;
        background: #f8fafc;
        border: 1px solid #e2e8f0;
        border-radius: 10px;
        color: #475569;
        font-size: 12px;
        line-height: 1.6;
        word-break: break-all;
    }

    .el-upload-list {
        margin-top: 8px;
    }

    .el-upload-list__item {
        border: 1px solid #e2e8f0;
        border-radius: 10px;
        margin-top: 10px !important;
        padding: 10px 12px;
        background: #ffffff;
        
        &:hover {
            background-color: #f8fafc;
        }

        .el-upload-list__item-name {
            margin: 0;
            font-size: 13px;
            color: #334155;
            font-weight: 500;
            display: flex;
            align-items: center;

            i {
                font-size: 16px;
                color: #64748b;
                margin-right: 8px;
            }
        }

        .el-progress {
            position: relative;
            top: 0;
            margin-top: 8px;
        }

        .el-progress-bar__outer {
            height: 6px !important;
            background-color: #f1f5f9 !important;
            border-radius: 3px;
        }

        .el-progress-bar__inner {
            background: linear-gradient(90deg, #3b82f6, #60a5fa) !important;
            border-radius: 3px;
        }

        .el-progress__text {
            font-size: 11px !important;
            color: #64748b;
            font-weight: 600;
        }
    }
}

::v-deep .premium-editor-dialog {
    border-radius: 16px;
    overflow: hidden;
    background: #ffffff;
    margin-top: 0 !important;
    box-shadow:
        0 25px 50px -12px rgba(0, 0, 0, 0.25),
        0 0 0 1px rgba(255, 255, 255, 0.5) inset;

    &.centered-dialog {
        position: absolute;
        top: 50%;
        left: 50%;
        transform: translate(-50%, -50%);
        margin: 0 !important;
    }

    .el-dialog__header {
        background: linear-gradient(180deg, #f8fafc 0%, #f1f5f9 100%);
        border-bottom: 1px solid #e2e8f0;
        padding: 14px 20px;

        .el-dialog__title {
            font-size: 14px;
            font-weight: 600;
            color: #0f172a;
            font-family: 'JetBrains Mono', 'Fira Code', monospace;
        }

        .el-dialog__headerbtn .el-dialog__close {
            color: #64748b;
            &:hover { color: #0f172a; }
        }
    }

    .el-dialog__body {
        padding: 0;
    }

    .editor-container {
        display: flex;
        background: #ffffff;
        min-height: 500px;
        max-height: 70vh;
        overflow-y: auto;
    }

    .line-numbers-gutter {
        width: 48px;
        background: #f8fafc;
        border-right: 1px solid #e2e8f0;
        padding-top: 16px;
        padding-bottom: 16px;
        flex-shrink: 0;
        text-align: right;
        user-select: none;
    }

    .line-number {
        font-family: 'JetBrains Mono', 'Fira Code', 'Courier New', monospace;
        font-size: 13px;
        line-height: 1.6;
        color: #94a3b8;
        padding-right: 12px;
    }

    .monaco-like-editor {
        flex: 1;

        .el-textarea__inner {
            border: none;
            border-radius: 0;
            font-family: 'JetBrains Mono', 'Fira Code', 'Courier New', monospace;
            font-size: 13px;
            line-height: 1.6;
            background: #ffffff !important;
            padding: 16px 20px;
            resize: none;
            color: #334155 !important;
            overflow: hidden !important;
            min-height: 500px !important;
            caret-color: #3b82f6;

            &::placeholder {
                color: #94a3b8;
            }

            &:focus {
                box-shadow: none;
            }
        }
    }

    .el-dialog__footer {
        padding: 12px 20px;
        border-top: 1px solid #f1f5f9;
        background: #f8fafc;
    }
}

::v-deep .premium-preview-dialog {
    border-radius: 16px;
    overflow: hidden;
    background: #ffffff;
    margin-top: 0 !important;
    box-shadow:
        0 25px 50px -12px rgba(0, 0, 0, 0.25),
        0 0 0 1px rgba(255, 255, 255, 0.5) inset;

    &.centered-dialog {
        position: absolute;
        top: 50%;
        left: 50%;
        transform: translate(-50%, -50%);
        margin: 0 !important;
    }

    .el-dialog__header {
        padding: 16px 20px;
        background: linear-gradient(180deg, #f8fafc 0%, #eef2ff 100%);
        border-bottom: 1px solid #e2e8f0;
    }

    .el-dialog__body {
        padding: 0;
        background: #0f172a;
    }

    .el-dialog__footer {
        padding: 12px 20px;
        border-top: 1px solid #e2e8f0;
        background: #f8fafc;
    }
}

.preview-stage {
    min-height: 520px;
    display: flex;
    align-items: center;
    justify-content: center;
    background:
        radial-gradient(circle at top, rgba(59, 130, 246, 0.14), transparent 24%),
        linear-gradient(180deg, #0f172a 0%, #111827 100%);
}

.preview-image,
.preview-video,
.preview-frame {
    width: 100%;
    min-height: 520px;
    max-height: 72vh;
    display: block;
    border: 0;
}

.preview-image {
    object-fit: contain;
}

.preview-video {
    background: #000000;
}

.preview-audio {
    width: min(560px, calc(100% - 48px));
    margin: 0 auto;
}
</style>
