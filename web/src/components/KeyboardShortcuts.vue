<template>
    <el-dialog
        :visible.sync="visible"
        width="480px"
        top="0"
        custom-class="shortcuts-dialog centered-dialog"
        :close-on-click-modal="true"
        append-to-body>
        <div slot="title" class="shortcuts-header">
            <i class="el-icon-info"></i>
            <span>{{ $t('keyboardShortcuts') || '键盘快捷键' }}</span>
        </div>
        <div class="shortcuts-content">
            <div class="shortcuts-section">
                <div class="section-title">{{ $t('workspaceShortcuts') || '工作区' }}</div>
                <div class="shortcut-item">
                    <div class="shortcut-keys">
                        <kbd>Alt</kbd> + <kbd>1</kbd>..<kbd>9</kbd>
                    </div>
                    <div class="shortcut-desc">{{ $t('switchTabByIndex') || '切换到第 N 个标签' }}</div>
                </div>
                <div class="shortcut-item">
                    <div class="shortcut-keys">
                        <kbd>Ctrl</kbd> + <kbd>Shift</kbd> + <kbd>W</kbd>
                    </div>
                    <div class="shortcut-desc">{{ $t('closeCurrentTab') || '关闭当前标签' }}</div>
                </div>
                <div class="shortcut-item">
                    <div class="shortcut-keys">
                        <kbd>Ctrl</kbd> + <kbd>Shift</kbd> + <kbd>F</kbd>
                    </div>
                    <div class="shortcut-desc">{{ $t('toggleFileBrowser') || '打开/关闭文件浏览器' }}</div>
                </div>
            </div>
            <div class="shortcuts-section">
                <div class="section-title">{{ $t('terminal') || '终端操作' }}</div>
                <div class="shortcut-item">
                    <div class="shortcut-keys">
                        <kbd>Ctrl</kbd> + <kbd>C</kbd>
                    </div>
                    <div class="shortcut-desc">{{ $t('copySelection') || '复制选中文本' }}</div>
                </div>
                <div class="shortcut-item">
                    <div class="shortcut-keys">
                        <kbd>Ctrl</kbd> + <kbd>V</kbd>
                    </div>
                    <div class="shortcut-desc">{{ $t('paste') || '粘贴' }}</div>
                </div>
                <div class="shortcut-item">
                    <div class="shortcut-keys">
                        <kbd>Ctrl</kbd> + <kbd>Shift</kbd> + <kbd>L</kbd>
                    </div>
                    <div class="shortcut-desc">{{ $t('clearTerminal') || '清屏（不影响远端）' }}</div>
                </div>
                <div class="shortcut-item">
                    <div class="shortcut-keys">
                        <kbd>Ctrl</kbd> + <kbd>+</kbd>
                    </div>
                    <div class="shortcut-desc">{{ $t('increaseFontSize') || '增大字体' }}</div>
                </div>
                <div class="shortcut-item">
                    <div class="shortcut-keys">
                        <kbd>Ctrl</kbd> + <kbd>-</kbd>
                    </div>
                    <div class="shortcut-desc">{{ $t('decreaseFontSize') || '减小字体' }}</div>
                </div>
                <div class="shortcut-item">
                    <div class="shortcut-keys">
                        <kbd>Ctrl</kbd> + <kbd>0</kbd>
                    </div>
                    <div class="shortcut-desc">{{ $t('resetFontSize') || '重置字体大小' }}</div>
                </div>
                <div class="shortcut-item">
                    <div class="shortcut-keys">
                        <kbd>Ctrl</kbd> + <kbd>滚轮</kbd>
                    </div>
                    <div class="shortcut-desc">{{ $t('zoomTerminal') || '缩放终端' }}</div>
                </div>
            </div>
            <div class="shortcuts-section">
                <div class="section-title">{{ $t('editFileTitle') || '文件编辑' }}</div>
                <div class="shortcut-item">
                    <div class="shortcut-keys">
                        <kbd>Ctrl</kbd> / <kbd>Cmd</kbd> + <kbd>S</kbd>
                    </div>
                    <div class="shortcut-desc">{{ $t('saveChanges') || '保存更改' }}</div>
                </div>
                <div class="shortcut-item">
                    <div class="shortcut-keys">
                        <kbd>Ctrl</kbd> / <kbd>Cmd</kbd> + <kbd>Z</kbd>
                    </div>
                    <div class="shortcut-desc">{{ $t('undoEdit') || '撤销修改' }}</div>
                </div>
                <div class="shortcut-item">
                    <div class="shortcut-keys">
                        <kbd>Ctrl</kbd> / <kbd>Cmd</kbd> + <kbd>Shift</kbd> + <kbd>Z</kbd>
                    </div>
                    <div class="shortcut-desc">{{ $t('redoEdit') || '重做修改' }}</div>
                </div>
                <div class="shortcut-item">
                    <div class="shortcut-keys">
                        <kbd>Ctrl</kbd> / <kbd>Cmd</kbd> + <kbd>Y</kbd>
                    </div>
                    <div class="shortcut-desc">{{ $t('redoEdit') || '重做修改' }}</div>
                </div>
            </div>
        </div>
    </el-dialog>
</template>

<script>
export default {
    name: 'KeyboardShortcuts',
    data() {
        return {
            visible: false
        }
    },
    methods: {
        show() {
            this.visible = true
        },
        hide() {
            this.visible = false
        }
    }
}
</script>

<style scoped lang="scss">
.shortcuts-header {
    display: flex;
    align-items: center;
    gap: 10px;
    font-size: 16px;
    font-weight: 700;
    color: #0f172a;

    i {
        font-size: 18px;
        color: #3b82f6;
    }
}

.shortcuts-content {
    padding: 8px 0;
}

.shortcuts-section {
    margin-bottom: 20px;

    &:last-child {
        margin-bottom: 0;
    }
}

.section-title {
    font-size: 13px;
    font-weight: 700;
    color: #334155;
    margin-bottom: 12px;
    padding-left: 4px;
    display: flex;
    align-items: center;
    gap: 8px;

    &::before {
        content: '';
        width: 3px;
        height: 14px;
        background: linear-gradient(135deg, #3b82f6, #7c3aed);
        border-radius: 2px;
        flex-shrink: 0;
    }
}

.shortcut-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 10px 12px;
    border-radius: 8px;
    transition: background 0.2s;

    &:hover {
        background: #f8fafc;
    }
}

.shortcut-keys {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 13px;
    color: #64748b;
    font-weight: 500;

    kbd {
        display: inline-block;
        padding: 4px 8px;
        font-family: 'SF Mono', 'Monaco', 'Consolas', monospace;
        font-size: 12px;
        font-weight: 600;
        color: #1e293b;
        background: linear-gradient(180deg, #ffffff 0%, #f1f5f9 100%);
        border: 1.5px solid #cbd5e1;
        border-radius: 6px;
        box-shadow: 0 2px 0 0 #cbd5e1, 0 1px 2px rgba(0, 0, 0, 0.05);
        line-height: 1;
        white-space: nowrap;
    }
}

.shortcut-desc {
    font-size: 13px;
    color: #475569;
    font-weight: 500;
}

::v-deep .shortcuts-dialog {
    border-radius: 16px;
    overflow: hidden;
    box-shadow: 0 20px 50px -12px rgba(0, 0, 0, 0.25);
    margin-top: 0 !important;

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
        padding: 18px 24px;
    }

    .el-dialog__body {
        padding: 20px 24px;
    }

    .el-dialog__headerbtn {
        top: 18px;
        right: 24px;
        font-size: 18px;

        .el-dialog__close {
            color: #64748b;
            font-weight: 700;

            &:hover {
                color: #3b82f6;
            }
        }
    }
}
</style>
