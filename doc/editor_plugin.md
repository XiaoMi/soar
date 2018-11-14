## Vim插件安装

* 首先安装Syntastic，安装方法参见[官方文档](https://github.com/vim-syntastic/syntastic#installation)
* 将`soar`二进制文件拷贝到可执行文件的查找路径($PATH)下，添加可执行权限`chmod a+x soar`
* 将doc/example/[soar.vim](http://github.com/XiaoMi/soar/raw/master/doc/example/soar.vim)文件拷贝至${SyntasticInstalledPath}/syntax_checkers/sql目录下
* 修改${SyntasticInstalledPath}/plugin/syntastic/registry.vim文件，增加sql文件的检查工具，`'sql':['soar', 'sqlint']`

### 插件演示

![Vim插件示例](https://raw.githubusercontent.com/XiaoMi/soar/master/doc/images/vim_plugin.png)

### 常见问题

#### 安装插件后无任何变化

安装了 Syntastic 没有任何显示，官方推荐通过如下配置来开启自动提示，不然用户无法看到SOAR给出的建议。

```vim
set statusline+=%#warningmsg#
set statusline+=%{SyntasticStatuslineFlag()}
set statusline+=%*

let g:syntastic_always_populate_loc_list = 1
let g:syntastic_auto_loc_list = 1
let g:syntastic_check_on_open = 1
let g:syntastic_check_on_wq = 0
```

如果soar二进制未在可执行文件查找路径下，或未添加可执行文件也会导致无法提供建议，可通过如下命令确认。

```bash
$ which soar
/usr/local/bin/soar
```
