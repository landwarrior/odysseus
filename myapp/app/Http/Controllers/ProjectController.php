<?php

namespace App\Http\Controllers;

use Illuminate\Http\Request;
use App\TrnProject;
use App\MstProcess;
use App\TrnProjectDetail;
use Illuminate\Support\Facades\Auth;
use Illuminate\Support\Facades\DB;
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
        $processes = MstProcess::all();
        $projectDetail = new TrnProjectDetail();
        return view(
            'project.create',
            ['project' => $project, 'processes' => $processes, 'details' => [$projectDetail]]
        );
    }

    public function store(ProjectRequest $request)
    {
        $user = Auth::user();
        if (!$user->is_admin) {
            return redirect(route('home'))->with('not_admin', '1');
        }
        DB::transaction(function () use ($request) {
            $project = new TrnProject();
            $project->project_no = $request->project_no;
            $project->name = $request->project_name;
            $project->order_amount = (int)$request->order_amount;
            $project->from_date = $request->from_date;
            $project->to_date = $request->to_date;
            $project->save();
            if ($request->details) {
                foreach ($request->details as $detail) {
                    $projectDetail = new TrnProjectDetail();
                    $projectDetail->project_no = $request['project_no'];
                    $projectDetail->process_id = $detail['process_id'];
                    $projectDetail->from_date = $detail['from_date'];
                    $projectDetail->to_date = $detail['to_date'];
                    $projectDetail->man_per_day = $detail['man_per_day'];
                    $projectDetail->pre_cost = $detail['pre_cost'];
                    $projectDetail->save();
                }
            }
        });

        return redirect('/project')->with('registered', '1');
    }

    public function edit($project_no)
    {
        $user = Auth::user();
        if (!$user->is_admin) {
            return redirect(route('home'))->with('not_admin', '1');
        }

        $project = TrnProject::findOrFail($project_no);
        $processes = MstProcess::all();
        $projectDetails = TrnProjectDetail::where('project_no', $project_no)->get();
        return view(
            'project.edit',
            ['project' => $project, 'processes' => $processes, 'details' => $projectDetails]
        );
    }

    public function update(ProjectRequest $request, $project_no)
    {
        $user = Auth::user();
        if (!$user->is_admin) {
            return redirect(route('home'))->with('not_admin', '1');
        }

        DB::transaction(function () use ($request, $project_no) {
            $project = TrnProject::findOrFail($project_no);
            $project->name = $request->project_name;
            $project->order_amount = $request->order_amount;
            $project->from_date = $request->from_date;
            $project->to_date = $request->to_date;
            $project->save();
            $projectDetails = TrnProjectDetail::where('project_no', $project_no)->delete();
            if ($request->details) {
                foreach ($request->details as $detail) {
                    $projectDetail = new TrnProjectDetail();
                    $projectDetail->project_no = $project_no;
                    $projectDetail->process_id = $detail['process_id'];
                    $projectDetail->from_date = $detail['from_date'];
                    $projectDetail->to_date = $detail['to_date'];
                    $projectDetail->man_per_day = (int)$detail['man_per_day'];
                    $projectDetail->pre_cost = (int)$detail['pre_cost'];
                    $projectDetail->save();
                }
            }
        });

        return redirect('/project')->with('registered', '1');
    }

    public function delete(Request $request, $project_no)
    {
        $user = Auth::user();
        if (!$user->is_admin) {
            return redirect(route('home'))->with('not_admin', '1');
        }

        DB::transaction(function () use ($project_no) {
            TrnProjectDetail::where('project_no', $project_no)->delete();
            TrnProject::find($project_no)->delete();
        });
        return redirect('/project')->with('deleted', '1');
    }
}
