<?php

namespace App\Http\Controllers;

use Illuminate\Http\Request;
use App\TrnProject;
use App\TrnProjectDetailHr;
use Illuminate\Support\Facades\Auth;

class ResultController extends Controller
{
    public function index()
    {
        $user = Auth::user();
        $targets = TrnProjectDetailHr::where('hr_cd', $user->hr_cd)->get();
        $p = [];
        foreach($targets as $target) {
            $p[] = $target->project_no;
        }
        $projects = TrnProject::whereIn('project_no', $p)->get();
        return view('result.index', ['projects' => $projects]);
    }
}
