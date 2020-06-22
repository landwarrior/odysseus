<?php

use Illuminate\Support\Facades\Route;

/*
|--------------------------------------------------------------------------
| Web Routes
|--------------------------------------------------------------------------
|
| Here is where you can register web routes for your application. These
| routes are loaded by the RouteServiceProvider within a group which
| contains the "web" middleware group. Now create something great!
|
*/
Auth::routes();

Route::get('/', 'HomeController@index')->name('home');

Route::get('/project', 'ProjectController@index');
Route::get('/project/create', 'ProjectController@create');
Route::post('/project', 'ProjectController@store');
Route::get('/project/{project_no}', 'ProjectController@edit');
Route::put('/project/{project_no}', 'ProjectController@update');
Route::delete('/project/{project_no}', 'ProjectController@delete');

Route::get('/hr', 'HrController@index');
Route::get('/hr/create', 'HrController@create');
Route::post('/hr', 'HrController@store');
Route::get('/hr/{hr_cd}', 'HrController@edit');
Route::put('/hr/{hr_cd}', 'HrController@update');
Route::delete('/hr/{hr_cd}', 'HrController@delete');
