<?php

namespace App\Http\Controllers;

use Illuminate\Http\Request;
use App\TrnProject;
use App\TrnProjectDetail;
use App\TrnHrResult;

class ActualController extends Controller
{
    /**
     * Create a new controller instance.
     *
     * @return void
     */
    public function __construct()
    {
        $this->middleware('auth');
    }

    public function index()
    {
      return view('actual.index');
    }
}
