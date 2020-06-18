<?php

namespace App\Http\Controllers;

use Illuminate\Http\Request;
use App\TrnProject;
use Illuminate\Support\Facades\Auth;
use App\Http\Requests\ProjectRequest;

class ProjectController extends Controller
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
        $user = Auth::user();
        if (!$user->is_admin) {
            return redirect(route('home'))->with('not_admin', '1');
        }
        $projects = TrnProject::all();
        return view('project.index', ['projects' => $projects]);
    }

    public function create()
    {
        $user = Auth::user();
        if (!$user->is_admin) {
            return redirect(route('home'))->with('not_admin', '1');
        }

        $project = new TrnProject();
        return view('project.create', ['project' => $project]);
    }

    public function store(ProjectRequest $request)
    {
        $user = Auth::user();
        if (!$user->is_admin) {
            return redirect(route('home'))->with('not_admin', '1');
        }
        $validated = $request->validated();
        $project = new TrnProject();
        $project->project_no = $request->project_no;
        $project->name = $request->project_name;
        $project->order_amount = $request->order_amount;
        $project->from_date = $request->from_date;
        $project->to_date = $request->to_date;
        $project->save();

        return redirect('/project');
    }
}
