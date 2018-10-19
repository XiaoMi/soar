"============================================================================
"File:        soar.vim
"Description: Syntax checking plugin for syntastic
"Maintainer:  Pengxiang Li<lipengxiang@xiaomi.com>
"License:     MIT
"============================================================================

if exists('g:loaded_syntastic_sql_soar_checker')
    finish
endif
let g:loaded_syntastic_sql_soar_checker= 1

let s:save_cpo = &cpo
set cpo&vim

function! SyntaxCheckers_sql_soar_GetLocList() dict
    let makeprg = self.makeprgBuild({
    \ 'args_after': '-report-type lint -query '})

    let errorformat = '%f:%l:%m'

    return SyntasticMake({
        \ 'makeprg': makeprg,
        \ 'errorformat': errorformat,
        \ 'defaults': {'type': 'W'},
        \ 'subtype': 'Style',
        \ 'returns': [0, 1] })
endfunction

call g:SyntasticRegistry.CreateAndRegisterChecker({
    \ 'filetype': 'sql',
    \ 'name': 'soar'})

let &cpo = s:save_cpo
unlet s:save_cpo

" vim: set sw=4 sts=4 et fdm=marker:
